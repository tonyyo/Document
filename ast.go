/*
Copyright 2017 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sqlparser

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/youtube/vitess/go/vt/vterrors"

	"flag"

	log "github.com/golang/glog"

	"strconv"

	"fmt"

	"math"

	"sort"

	"github.com/youtube/vitess/go/errors"
	"github.com/youtube/vitess/go/sqltypes"
	"github.com/youtube/vitess/go/utils/schemautil"
	querypb "github.com/youtube/vitess/go/vt/proto/query"
	vtrpcpb "github.com/youtube/vitess/go/vt/proto/vtrpc"
)

const (
	createDatabaseTemplate          string = "CREATE DATABASE IF NOT EXISTS %s"
	createGlobalUniqueIndexTemplate string = "CREATE GLOBAL UNIQUE INDEX %s ON %s(%s)"
	dropIndexTemplate               string = "DROP INDEX %s ON %s.%s"
	// can generate create/drpo server every where
	InvalidNumOfArguments string = "ORA-00909: invalid number of arguments"
	dropServerTemplate    string = "DROP SERVER IF EXISTS %s"
	createServerTemplate  string = "CREATE SERVER %s FOREIGN DATA WRAPPER VITESS OPTIONS(USER '%s', PASSWORD '%s', HOST '%s', PORT %d, DATABASE %s)"

	// MixedGateMySQLErr works for not to route queries to mfed
	MixedGateMySQLErr = "query contains function that gate not support mixed with that gate support"

	// add alias for derived table
	derivedTableAliasFormat string = "kundb_derived_table%d"

	// suppport rownum
	rnAutoGenAliasFormat  string = "kundb_rownum%d"
	rnSelectItemFormat    string = "@rownum%d:=@rownum%d+1"
	rnGenerateTableFormat string = "(select @rownum%d:=0) as %s"

	// RNRoutedtoMfed   string = "rownum will be route to mfed"
	RNTooComplicate  string = "rownum sql is too complicated"
	RNWithLimitErr   string = "rownum with limit is not support"
	RNUpdateSetError string = "set rownum in update statement is not support"

	TableExistsComment string = "/*!99999 select for mfed ddl*/"
)

var (
	// parserPool is a pool for parser objects.
	parserPool = sync.Pool{}
	// zeroParser is a zero-initialized parser to help reinitialize the parser for pooling.
	zeroParser          = *(yyNewParser().(*yyParserImpl))
	lowerCaseTableNames = flag.Bool("lower_case_table_names", false, "Table names are stored in lowercase and name comparisons are not case sensitive. Kundb converts all table names to lowercase on storage and lookup. This behavior also applies to database names and table aliases.")
	// FormatCheckCOnstraint is a switch for output check constraint information
	FormatCheckConstraint = false
)

// Init settings
// Caution: this function must call after flag.Parse()
func Init() {
	if *lowerCaseTableNames {
		log.Info("set case sesentive to false")
		schemautil.SetTNCS(false)
		schemautil.SetDNCS(false)
	}
}

// Search min and max continuously
// for example:
// [0, 1, 2, 3, 5, 6] returns 1,3
// [-1, 0, 2, 3, 4, 5, 10, 11] returns 2,5
// [-2, -1] returns 0, 0
// [] returns 0,0
// [1] returns 1, 1
func SearchMin(in []int) (int, int) {
	var min, max int
	sort.Ints(in)
	for _, item := range in {
		if item <= 0 {
			continue
		}
		if min == 0 {
			min = item
			max = item
		}
		if item == max+1 {
			max = item
		}
	}
	return min, max
}

// RNLeftOrRight means the position of rownum in comparison expr, in left or right?
type RNLeftOrRight int

const (
	RNUnknown RNLeftOrRight = iota
	RNinLeft
	RNinRight
	RNinBoth
)

// genDerivedTableAlias generates a alias name
func genDerivedTableAlias(i int64) string {
	return fmt.Sprintf(derivedTableAliasFormat, i)
}

// genRownumAlias generates a alias name
func genRownumAlias(i int64) string {
	return fmt.Sprintf(rnAutoGenAliasFormat, i)
}

func genRownumVirtualTbl(rnIndex int) string {
	return fmt.Sprintf(rnGenerateTableFormat, rnIndex, genRownumAlias(int64(rnIndex)))
}

func parseDefaultAndNull(vals []DefaultNullVal) (defaultVal *SQLVal, notNull BoolVal) {
	for _, val := range vals {
		switch node := val.(type) {
		case *SQLVal:
			defaultVal = node
		case BoolVal:
			notNull = node
		default:
		}
	}
	return
}

// GetLowerCaseTableNames return the value of lowerCaseTableNames
func GetLowerCaseTableNames() bool {
	return *lowerCaseTableNames
}

// IsCharType return true if the give typ is list here
func IsCharType(typ string) bool {
	switch strings.ToLower(typ) {
	case "char", "varchar", "binary", "varbinary", "text", "tinytext", "mediumtext", "longtext", "clob", "blob", "tinyblob", "mediumbolb", "longblob", "json", "enum":
		return true
	default:
		return false
	}
}

// IsNumericFunc return true if the shardFunc call ToUint64 function
func IsNumericFunc(shardFunc string) bool {
	switch strings.ToLower(shardFunc) {
	case "hash", "numeric", "lookup_hash", "lookup_hash_unique":
		return true
	default:
		return false
	}
}

func newMatrixKey(typ, shardFunc string) string {
	return typ + "|||" + shardFunc
}

// typSfMatrix is matrix of type and shard function
// add some special case
// bool value shows the the compatibility of type and function
var typSfMatrix = map[string]bool{
	newMatrixKey("decimal", "binary_md5"): false,
	newMatrixKey("decimal", "hash"):       false,
}

// IsCompatible return content in typSfMatrix by giving column type and shard function
// if key not exists in typSfMatrix, return false
func IsCompatible(colType, shardFunc string) bool {
	typ, sf := strings.ToLower(colType), strings.ToLower(shardFunc)
	res, ok := typSfMatrix[newMatrixKey(typ, sf)]
	if !ok {
		return true
	}
	return res
}

// GenCreateDBSQL generate sql to create database
func GenCreateDBSQL(serverName string) string {
	buf := NewTrackedBuffer(nil)
	FormatID(buf, serverName, strings.ToLower(serverName), true)
	return fmt.Sprintf(createDatabaseTemplate, buf.String())
}

func GenGlobalUniqueIndexSQL(indexName, tbName, priName string) string {
	buf := NewTrackedBuffer(nil)
	FormatID(buf, indexName, strings.ToLower(indexName), true)
	return fmt.Sprintf(createGlobalUniqueIndexTemplate, buf.String(), tbName, priName)
}

func GenDropIndexSQL(indexName, schema, tbName string) string {
	buf := NewTrackedBuffer(nil)
	FormatID(buf, indexName, strings.ToLower(indexName), true)
	return fmt.Sprintf(dropIndexTemplate, buf.String(), schema, tbName)
}

// GenDropServerSQL generate sql to drop server
func GenDropServerSQL(serverName string) string {
	buf := NewTrackedBuffer(nil)
	FormatID(buf, serverName, strings.ToLower(serverName), true)
	dbF1 := buf.String()
	dbF1 = strings.ReplaceAll(dbF1, "@", "AT")
	return fmt.Sprintf(dropServerTemplate, dbF1)
}

// GenCreateServerSQL generate sql to create server
func GenCreateServerSQL(dbName, user, passwd, host string, port int) string {
	buf := NewTrackedBuffer(nil)
	FormatID(buf, dbName, strings.ToLower(dbName), false)
	// form1 of database name
	dbF1 := buf.String()
	buf.Reset()
	FormatIDUniform(buf, dbName, true)
	// form2 of database name
	dbF2 := buf.String()
	log.V(2).Infof("two forms are %s, %s", dbF1, dbF2)
	// why we need different forms here?
	// cause database in create server options only allows uniform format
	dbF1 = strings.ReplaceAll(dbF1, "@", "AT")
	return fmt.Sprintf(createServerTemplate, dbF1, user, passwd, host, port, dbF2)
}

// SetLowerCaseTableNames set the value of lowerCaseTableNames
func SetLowerCaseTableNames(value bool) {
	*lowerCaseTableNames = value
	schemautil.SetTNCS(!*lowerCaseTableNames)
	schemautil.SetDNCS(!*lowerCaseTableNames)
}

// RemoveBackendHints works for removing backend hints
func RemoveBackendHints(stmt Statement) string {
	// remove backend hint
	buf := NewTrackedBuffer(func(buf *TrackedBuffer, node SQLNode) {
		switch node := node.(type) {
		case Hints:
			return
		default:
			node.Format(buf)
			return
		}
	})
	buf.Myprintf("%v", stmt)
	return buf.String()
}

// yyParsePooled is a wrapper around yyParse that pools the parser objects. There isn't a
// particularly good reason to use yyParse directly, since it immediately discards its parser.  What
// would be ideal down the line is to actually pool the stacks themselves rather than the parser
// objects, as per https://github.com/cznic/goyacc/blob/master/main.go. However, absent an upstream
// change to goyacc, this is the next best option.
//
// N.B: Parser pooling means that you CANNOT take references directly to parse stack variables (e.g.
// $$ = &$4) in sql.y rules. You must instead add an intermediate reference like so:
//    showCollationFilterOpt := $4
//    $$ = &Show{Type: string($2), ShowCollationFilterOpt: &showCollationFilterOpt}
func yyParsePooled(yylex yyLexer) int {
	// Being very particular about using the base type and not an interface type b/c we depend on
	// the implementation to know how to reinitialize the parser.
	var parser *yyParserImpl
	i := parserPool.Get()
	if i != nil {
		parser = i.(*yyParserImpl)
	} else {
		parser = yyNewParser().(*yyParserImpl)
	}
	defer func() {
		*parser = zeroParser
		parserPool.Put(parser)
	}()
	return parser.Parse(yylex)
}

// Instructions for creating new types: If a type
// needs to satisfy an interface, declare that function
// along with that interface. This will help users
// identify the list of types to which they can assert
// those interfaces.
// If the member of a type has a string with a predefined
// list of values, declare those values as const following
// the type.
// For interfaces that define dummy functions to consolidate
// a set of types, define the function as iTypeName.
// This will help avoid name collisions.

// ParseExpr, so tricky, optimize me
func ParseExpr(expr string) (Expr, error) {
	if expr == "" {
		return nil, errors.New(errors.ErrInvalidArgument, "cant parse empty expr")
	}
	sql := "SELECT " + expr
	stmt, err := Parse(sql)
	if err != nil {
		return nil, err
	}
	if sel, ok := stmt.(SelectStatement); ok {
		if s, ok := sel.(*Select); ok {
			if len(s.SelectExprs) != 1 {
				return nil, fmt.Errorf("select should have one expression")
			}
			expr, ok := s.SelectExprs[0].(*AliasedExpr)
			if !ok {
				return nil, fmt.Errorf("select expr should be aliased expr")
			}
			return expr.Expr, nil
		} else {
			return nil, fmt.Errorf("select statement should be select")
		}
	}
	return nil, fmt.Errorf("stmt of should be select statement")
}

// Parse parses the sql and returns a Statement, which
// is the AST representation of the query. If a DDL statement
// is partially parsed but still contains a syntax error, the
// error is ignored and the DDL is returned anyway.
func Parse(sql string) (Statement, error) {
	tokenizer := NewStringTokenizer(sql)
	if yyParsePooled(tokenizer) != 0 {
		if tokenizer.partialDDL != nil {
			log.Warningf("ignoring error parsing DDL '%s': %v", sql, tokenizer.LastError)
			tokenizer.ParseTree = tokenizer.partialDDL
			return tokenizer.ParseTree, nil
		}
		return nil, vterrors.New(vtrpcpb.Code_INVALID_ARGUMENT, tokenizer.LastError.Error())
	}
	return tokenizer.ParseTree, nil
}

// ParseStrictDDL is the same as Parse except it errors on
// partially parsed DDL statements.
func ParseStrictDDL(sql string) (Statement, error) {
	tokenizer := NewStringTokenizer(sql)
	if yyParsePooled(tokenizer) != 0 {
		return nil, tokenizer.LastError
	}
	return tokenizer.ParseTree, nil
}

// ParseExtra is the same as Parse except it returns some extra information
func ParseExtra(sql string) (Statement, ExtraParseInfo, error) {
	tokenizer := NewStringTokenizer(sql)
	if yyParsePooled(tokenizer) != 0 {
		if tokenizer.partialDDL != nil {
			log.Warningf("ignoring error parsing DDL '%s': %v", sql, tokenizer.LastError)
			tokenizer.ParseTree = tokenizer.partialDDL
			return tokenizer.ParseTree, tokenizer.ExtraParseInfo, nil
		}
		return nil, ExtraParseInfo{}, vterrors.New(vtrpcpb.Code_INVALID_ARGUMENT, tokenizer.LastError.Error())
	}
	return tokenizer.ParseTree, tokenizer.ExtraParseInfo, nil
}

// SplitStatementToPieces split raw sql statement that may have multi sql pieces to sql pieces
// returns the sql pieces blob contains; or error if sql cannot be parsed
func SplitStatementToPieces(blob string) (pieces []string, err error) {
	pieces = make([]string, 0, 16)
	tokenizer := NewStringTokenizer(blob)

	tkn := 0
	var stmt string
	stmtBegin := 0
	for {
		tkn, _ = tokenizer.Scan()
		if tkn == ';' {
			stmt = blob[stmtBegin : tokenizer.Position-2]
			pieces = append(pieces, stmt)
			stmtBegin = tokenizer.Position - 1

		} else if tkn == 0 || tkn == eofChar {
			blobTail := tokenizer.Position - 2

			if stmtBegin < blobTail {
				stmt = blob[stmtBegin : blobTail+1]
				pieces = append(pieces, stmt)
			}
			break
		}
	}
	if tokenizer.LastError != nil {
		err = fmt.Errorf("unsupported action %v for sequence", tokenizer.LastError)
	}

	return
}

// SQLNode defines the interface for all nodes
// generated by the parser.
type SQLNode interface {
	Format(buf *TrackedBuffer)
	// WalkSubtree calls visit on all underlying nodes
	// of the subtree, but not the current one. Walking
	// must be interrupted if visit returns an error.
	WalkSubtree(visit Visit) error
}

// Visit defines the signature of a function that
// can be used to visit all nodes of a parse tree.
type Visit func(node SQLNode) (kontinue bool, err error)

// Walk calls visit on every node.
// If visit returns true, the underlying nodes
// are also visited. If it returns an error, walking
// is interrupted, and the error is returned.
func Walk(visit Visit, nodes ...SQLNode) error {
	for _, node := range nodes {
		if node == nil {
			continue
		}
		kontinue, err := visit(node)
		if err != nil {
			return err
		}
		if kontinue {
			err = node.WalkSubtree(visit)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// String returns a string representation of an SQLNode.
func String(node SQLNode) string {
	if node == nil {
		return ""
	}
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

// Stringf returns a string representation of an SQLNode.
func Stringf(node SQLNode, nf NodeFormatter) string {
	if node == nil {
		return ""
	}
	buf := NewTrackedBuffer(nf)
	buf.Myprintf("%v", node)
	return buf.String()
}

// Append appends the SQLNode to the buffer.
func Append(buf *bytes.Buffer, node SQLNode) {
	tbuf := &TrackedBuffer{
		Buffer: buf,
	}
	node.Format(tbuf)
}

// Statement represents a statement.
type Statement interface {
	iStatement()
	SQLNode
}

func (*Union) iStatement()      {}
func (*Select) iStatement()     {}
func (*Insert) iStatement()     {}
func (*Update) iStatement()     {}
func (*Delete) iStatement()     {}
func (*Set) iStatement()        {}
func (*DDL) iStatement()        {}
func (*Show) iStatement()       {}
func (*Use) iStatement()        {}
func (*Begin) iStatement()      {}
func (*Commit) iStatement()     {}
func (*Rollback) iStatement()   {}
func (*Explain) iStatement()    {}
func (*OtherRead) iStatement()  {}
func (*OtherAdmin) iStatement() {}
func (*Call) iStatement()       {}
func (*DCL) iStatement()        {}

// ParenSelect can actually not be a top level statement,
// but we have to allow it because it's a requirement
// of SelectStatement.
func (*ParenSelect) iStatement() {}

// SelectStatement any SELECT statement.
type SelectStatement interface {
	iSelectStatement()
	iStatement()
	iInsertRows()
	AddOrder(*Order)
	SetLimit(*Limit)
	GetColumnVal(index int) ([]int, error)
	GetBindVarKey(index int) ([]string, error)
	SQLNode
}

func (*Select) iSelectStatement()      {}
func (*Union) iSelectStatement()       {}
func (*ParenSelect) iSelectStatement() {}

// Select represents a SELECT statement.
type Select struct {
	Cache        string
	Comments     Comments
	Distinct     string
	Hints        string
	ExtraHints   Hints
	SelectExprs  SelectExprs
	From         TableExprs
	Where        *Where
	GroupBy      GroupBy
	Having       *Where
	OrderBy      OrderBy
	Limit        *Limit
	Lock         string
	Hierarchical *HierarchicalQueryClause
	IntoClause   *IntoClause
}

type HierarchicalQueryClause struct {
	IsStartFirstly bool
	NocycleOpt     string
	StartPartOpt   Expr
	Condition      Expr
}

func (node *HierarchicalQueryClause) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	if !node.IsStartFirstly {
		buf.Myprintf(" CONNECT BY")
		if node.NocycleOpt != "" {
			buf.Myprintf(" NOCYCLE")
		}
		buf.Myprintf(" %v ", node.Condition)
		if node.StartPartOpt != nil {
			buf.Myprintf("START WITH %v ", node.StartPartOpt)
		}
	} else {
		buf.Myprintf(" START WITH %v", node.StartPartOpt)
		buf.Myprintf(" CONNECT BY")
		if node.NocycleOpt != "" {
			buf.Myprintf(" NOCYCLE")
		}
		buf.Myprintf(" %v ", node.Condition)
	}
}

func (node *HierarchicalQueryClause) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.StartPartOpt,
	)
}

// Select.Distinct
const (
	DistinctStr      = "distinct "
	StraightJoinHint = "straight_join "
)

// Select.Lock
const (
	ForUpdateStr = " for update"
	ShareModeStr = " lock in share mode"
)

// Select.Cache
const (
	SQLCacheStr   = "sql_cache "
	SQLNoCacheStr = "sql_no_cache "
)

// Hints represents Hint list.
type Hints []Hint

// Format formats the node.
func (node Hints) Format(buf *TrackedBuffer) {
	for _, n := range node {
		buf.Myprintf("%v", n)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node Hints) WalkSubtree(visit Visit) error {
	for _, n := range node {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}
	return nil
}

// Hint represents a hint.
type Hint interface {
	IHint()
	SQLNode
}

// IHint identifies interface Hint.
func (*BackendHint) IHint() {}

// IHint identifies interface Hint.
func (*GlkjoinHint) IHint() {}

// Backend.
const (
	Inceptor = "inceptor "
	Mfed     = "mfed "
)

// BackendHint hints what backend to use.
type BackendHint struct {
	Backend string
}

// Format formats the node.
func (node *BackendHint) Format(buf *TrackedBuffer) {
}

// WalkSubtree walks the nodes of the subtree.
func (node *BackendHint) WalkSubtree(visit Visit) error {
	return nil
}

// MarshalJSON marshals into JSON.
func (node *BackendHint) MarshalJSON() ([]byte, error) {
	return json.Marshal("backend " + node.Backend)
}

// GlkjoinHint hints to use global lookup join in the underlying backend.
type GlkjoinHint struct {
	TableNames TableNames
}

// Format formats the node.
func (node *GlkjoinHint) Format(buf *TrackedBuffer) {
}

// WalkSubtree walks the nodes of the subtree.
func (node *GlkjoinHint) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.TableNames,
	)
}

// MarshalJSON marshals into JSON.
func (node *GlkjoinHint) MarshalJSON() ([]byte, error) {
	return json.Marshal("glkjoin(" + String(node.TableNames) + ")")
}

// AddOrder adds an order by element
func (node *Select) AddOrder(order *Order) {
	node.OrderBy = append(node.OrderBy, order)
}

// SetLimit sets the limit clause
func (node *Select) SetLimit(limit *Limit) {
	node.Limit = limit
}

// GetColumnVal returns value of column in insert/update
func (node *Select) GetColumnVal(index int) ([]int, error) {
	return nil, errors.New(errors.ErrNotSupport, "not support")
}

func (node *Select) GetBindVarKey(index int) ([]string, error) {
	return nil, errors.New(errors.ErrNotSupport, "not support")
}

func (node *Select) formatRownumNoOrderBy(buf *TrackedBuffer, countSel, countWhere int) {
	// not rownum
	if countSel <= 0 && countWhere <= 0 {
		return
	}
	rownumStart := buf.rownumIndex
	buf.Myprintf("select %v%s%s%s%v", node.Comments, node.Cache, node.Distinct, node.Hints, node.ExtraHints)
	buf.Myprintf("%v", node.SelectExprs)
	// ugly, add a limit to first subquery
	switch te := node.From[0].(type) {
	case *AliasedTableExpr:
		if sub, ok := te.Expr.(*Subquery); ok {
			switch sel := sub.Select.(type) {
			case *Select:
				limit := sel.Limit
				if limit == nil {
					sel.Limit = NewLimit(0, math.MaxInt64)
				}
			}
		}
	}
	buf.Myprintf(" from %v", node.From)
	delimiter := ","
	for i := rownumStart; i < rownumStart+countSel; i++ {
		buf.Myprintf("%s%s", delimiter, genRownumVirtualTbl(i))
	}
	if countWhere > 0 {
		// if the node has limit and rownum same time, then original limit is not formatted
		newWhere, removed, err := node.Where.RemoveRownum()
		if err != nil {
			log.Errorf("failed to remove rownum, err is %s", err.Error())
			return
		}
		limit, err := ConvertToLimit(removed)
		if err != nil {
			log.Errorf("failed to convert to limit, err is %s", err.Error())
			return
		}
		buf.Myprintf("%v%v%v%v%s", newWhere, node.GroupBy, node.Having, limit, node.Lock)
		return
	}
	buf.Myprintf("%v%v%v%v%s", node.Where, node.GroupBy, node.Having, node.Limit, node.Lock)
}

// Format formats the node.
func (node *Select) Format(buf *TrackedBuffer) {
	countSel, countWhere := CountRownum(node.SelectExprs), CountRownum(node.Where)
	if (countSel > 0 || countWhere > 0) && len(node.OrderBy) > 0 {
		// pull up order by
		// make rownum before order by
		// aim at: select rownum,t.* from t order by xxx
		buf.Myprintf("select * from (")
		node.formatRownumNoOrderBy(buf, countSel, countWhere)
		buf.Myprintf(") %s%v", genRownumAlias(1), node.OrderBy)
	} else if countSel > 0 || countWhere > 0 {
		node.formatRownumNoOrderBy(buf, countSel, countWhere)
	} else {
		buf.Myprintf("select %v%s%s%s%v%v%v from %v%v%v%v%v%v%v%s",
			node.Comments, node.Cache, node.Distinct, node.Hints, node.ExtraHints, node.SelectExprs, node.IntoClause,
			node.From, node.Where, node.Hierarchical,
			node.GroupBy, node.Having, node.OrderBy,
			node.Limit, node.Lock)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *Select) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Comments,
		node.SelectExprs,
		node.From,
		node.Where,
		node.GroupBy,
		node.Having,
		node.OrderBy,
		node.Limit,
		node.Hierarchical,
		node.IntoClause,
	)
}

// AddWhere adds the boolean expression to the
// WHERE clause as an AND condition. If the expression
// is an OR clause, it parenthesizes it. Currently,
// the OR operator is the only one that's lower precedence
// than AND.
func (node *Select) AddWhere(expr Expr) {
	if _, ok := expr.(*OrExpr); ok {
		expr = &ParenExpr{Expr: expr}
	}
	if node.Where == nil {
		node.Where = &Where{
			Type: TypWhere,
			Expr: expr,
		}
		return
	}
	node.Where.Expr = &AndExpr{
		Left:  node.Where.Expr,
		Right: expr,
	}
}

// AddHaving adds the boolean expression to the
// HAVING clause as an AND condition. If the expression
// is an OR clause, it parenthesizes it. Currently,
// the OR operator is the only one that's lower precedence
// than AND.
func (node *Select) AddHaving(expr Expr) {
	if _, ok := expr.(*OrExpr); ok {
		expr = &ParenExpr{Expr: expr}
	}
	if node.Having == nil {
		node.Having = &Where{
			Type: TypHaving,
			Expr: expr,
		}
		return
	}
	node.Having.Expr = &AndExpr{
		Left:  node.Having.Expr,
		Right: expr,
	}
}

// ParenSelect is a parenthesized SELECT statement.
type ParenSelect struct {
	Select SelectStatement
}

// AddOrder adds an order by element
func (node *ParenSelect) AddOrder(order *Order) {
	panic("unreachable")
}

// SetLimit sets the limit clause
func (node *ParenSelect) SetLimit(limit *Limit) {
	panic("unreachable")
}

// Format formats the node.
func (node *ParenSelect) Format(buf *TrackedBuffer) {
	buf.Myprintf("(%v)", node.Select)
}

// WalkSubtree walks the nodes of the subtree.
func (node *ParenSelect) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Select,
	)
}

// GetColumnVal returns value of column in insert/update
func (node *ParenSelect) GetColumnVal(index int) ([]int, error) {
	return nil, errors.New(errors.ErrNotSupport, "not support")
}

func (node *ParenSelect) GetBindVarKey(index int) ([]string, error) {
	return nil, errors.New(errors.ErrNotSupport, "not support")
}

// Union represents a UNION statement.
type Union struct {
	Type        string
	Left, Right SelectStatement
	OrderBy     OrderBy
	Limit       *Limit
	Lock        string
}

// Union.Type
const (
	UnionStr         = "union"
	UnionAllStr      = "union all"
	UnionDistinctStr = "union distinct"
)

// AddOrder adds an order by element
func (node *Union) AddOrder(order *Order) {
	node.OrderBy = append(node.OrderBy, order)
}

// SetLimit sets the limit clause
func (node *Union) SetLimit(limit *Limit) {
	node.Limit = limit
}

// GetColumnVal returns value of column in insert/update
func (node *Union) GetColumnVal(index int) ([]int, error) {
	return nil, errors.New(errors.ErrNotSupport, "not support")
}

func (node *Union) GetBindVarKey(index int) ([]string, error) {
	return nil, errors.New(errors.ErrNotSupport, "not support")
}

// Format formats the node.
func (node *Union) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v %s %v%v%v%s", node.Left, node.Type, node.Right,
		node.OrderBy, node.Limit, node.Lock)
}

// WalkSubtree walks the nodes of the subtree.
func (node *Union) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Left,
		node.Right,
	)
}

// Insert represents an INSERT or REPLACE statement.
// Per the MySQL docs, http://dev.mysql.com/doc/refman/5.7/en/replace.html
// Replace is the counterpart to `INSERT IGNORE`, and works exactly like a
// normal INSERT except if the row exists. In that case it first deletes
// the row and re-inserts with new values. For that reason we keep it as an Insert struct.
// Replaces are currently disallowed in sharded schemas because
// of the implications the deletion part may have on vindexes.
type Insert struct {
	Action   string
	Comments Comments
	Ignore   string
	Table    TableName
	Columns  Columns
	Rows     InsertRows
	OnDup    OnDup
}

// DDL strings.
const (
	InsertStr  = "insert"
	ReplaceStr = "replace"
)

// Format formats the node.
func (node *Insert) Format(buf *TrackedBuffer) {
	buf.Myprintf("%s %v%sinto %v%v %v%v",
		node.Action,
		node.Comments, node.Ignore,
		node.Table, node.Columns, node.Rows, node.OnDup)
}

// WalkSubtree walks the nodes of the subtree.
func (node *Insert) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Comments,
		node.Table,
		node.Columns,
		node.Rows,
		node.OnDup,
	)
}

// GetColumnVal return the values of insert statement
// this function need columns
func (node *Insert) GetColumnVal(name string) ([]int, error) {
	index, err := node.GetColumnIndex(name)
	if err != nil {
		return nil, err
	}
	return node.Rows.GetColumnVal(index)
}

func (node *Insert) GetColumnBindVal(name string, bindVar map[string]*querypb.BindVariable) ([]int, error) {
	var res []int
	index, err := node.GetColumnIndex(name)
	if err != nil {
		return nil, err
	}
	bvks, err := node.Rows.GetBindVarKey(index)
	if err != nil {
		return nil, err
	}
	for _, bvk := range bvks {
		bvk = strings.TrimPrefix(bvk, ":")
		val, ok := bindVar[bvk]
		if !ok {
			return nil, fmt.Errorf("no data in bind variables for key[%s]", bvk)
		}
		if !sqltypes.IsIntegral(val.GetType()) {
			return nil, errors.New(errors.ErrNotSupport, "only support int type")
		}
		intV, err := strconv.Atoi(string(val.GetValue()))
		if err != nil {
			return nil, errors.New(errors.ErrUnexpected, fmt.Sprintf("failed to get int value, err is %s", err.Error()))
		}
		res = append(res, intV)
	}
	return res, nil
}

// GetColumnIndex return the index of one column in insert statement
// this function need columns
func (node *Insert) GetColumnIndex(name string) (int, error) {
	if node.Columns == nil {
		return -1, errors.New(errors.ErrInvalidArgument, "no insert columns")
	}
	if node.Rows == nil {
		return -1, errors.New(errors.ErrInvalidArgument, "no insert values")
	}
	index := node.Columns.FindColumn(NewColIdent(name))
	return index, nil
}

// InsertRows represents the rows for an INSERT statement.
type InsertRows interface {
	iInsertRows()
	SQLNode
	GetColumnVal(index int) ([]int, error)
	GetBindVarKey(index int) ([]string, error)
}

func (*Select) iInsertRows()      {}
func (*Union) iInsertRows()       {}
func (Values) iInsertRows()       {}
func (*ParenSelect) iInsertRows() {}

// Update represents an UPDATE statement.
type Update struct {
	Comments   Comments
	TableExprs TableExprs
	Exprs      UpdateExprs
	Where      *Where
	OrderBy    OrderBy
	Limit      *Limit
}

// Format formats the node.
func (node *Update) Format(buf *TrackedBuffer) {
	buf.Myprintf("update %v%v set %v%v%v%v",
		node.Comments, node.TableExprs,
		node.Exprs, node.Where, node.OrderBy, node.Limit)
}

// WalkSubtree walks the nodes of the subtree.
func (node *Update) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Comments,
		node.TableExprs,
		node.Exprs,
		node.Where,
		node.OrderBy,
		node.Limit,
	)
}

// Delete represents a DELETE statement.
type Delete struct {
	Comments   Comments
	Targets    TableNames
	TableExprs TableExprs
	Where      *Where
	OrderBy    OrderBy
	Limit      *Limit
}

// Format formats the node.
func (node *Delete) Format(buf *TrackedBuffer) {
	buf.Myprintf("delete %v", node.Comments)
	if node.Targets != nil {
		buf.Myprintf("%v ", node.Targets)
	}
	buf.Myprintf("from %v%v%v%v", node.TableExprs, node.Where, node.OrderBy, node.Limit)
}

// WalkSubtree walks the nodes of the subtree.
func (node *Delete) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Comments,
		node.Targets,
		node.TableExprs,
		node.Where,
		node.OrderBy,
		node.Limit,
	)
}

// Set represents a SET statement.
type Set struct {
	Comments Comments
	Exprs    SetExprs
	Scope    string
}

// Set.Scope or Show.Scope
const (
	SessionScope  = "session"
	GlobalScope   = "global"
	ImplicitScope = ""
)

// Format formats the node.
func (node *Set) Format(buf *TrackedBuffer) {
	if node.Scope == "" {
		buf.Myprintf("set %v%v", node.Comments, node.Exprs)
	} else {
		buf.Myprintf("set %v%s %v", node.Comments, node.Scope, node.Exprs)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *Set) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Comments,
		node.Exprs,
	)
}

type ddlType int

// ddl types
const (
	// Und is the default ddl type
	Und ddlType = iota
	SchemaDDL
	TableDDL
	ViewDDL
	IndexDDL
	SequenceDDL
	TriggerDDL
	FunctionDDL
	ProcedureDDL
	ServerDDL
	SynonymDDL
	PackageDDL
)

const (
	// HashPartition defines the hash partition
	HashPartition = "hash"
	// ReplicationPartition define the replication partition
	ReplicationPartition = "replication"
	// RangePartition defines the type of range partition
	RangePartition = "range"
	// ListPartition defines the type of list partition
	ListPartition = "list"
	// IntervalPartition defines the type of interval partition
	IntervalPartition = "interval"
	// Reference defines the type of reference partition
	ReferencePartition = "reference"
)

type AlterTyp int

const (
	AlterUnknown AlterTyp = iota
	AlterAdd
	AlterDrop
	AlterModify
	AlterRename
)

func (at AlterTyp) String() string {
	if at == AlterUnknown {
		return "AlterUnknown"
	} else if at == AlterAdd {
		return "AlterAdd"
	} else if at == AlterDrop {
		return "AlterDrop"
	} else if at == AlterModify {
		return "AlterModify"
	} else if at == AlterRename {
		return "AlterRename"
	}
	return ""
}

type AlterObject struct {
	ObjTyp  AlterObjTyp
	ObjName string
}

func NewAlterObject(objTyp AlterObjTyp, name string) AlterObject {
	return AlterObject{
		ObjTyp:  objTyp,
		ObjName: name,
	}
}

type AlterObjTyp int

const (
	UnknowObj AlterObjTyp = iota
	ColumnObj
	ConstraintObj
	FullTextObj
	IndexObj
	KeyObj
	PrimaryObj
	SpatialObj
	UniqueObj
)

// DDL represents a CREATE, ALTER, DROP, RENAME, Truncate statement.
// Table is set for AlterStr, DropStr, RenameStr, AlterRenameStr, TruncateStr
// NewName is set for AlterStr, CreateStr, RenameStr, AlterRenameStr, TruncateStr.
type DDL struct {
	Action             string
	AlterTyp           AlterTyp
	AlterObject        AlterObject
	NewName            TableName
	Table              TableName
	Comments           Comments
	IfTemporary        bool
	IfExists           bool
	IfNotExists        bool
	CreateOrReplace    bool
	TableSpec          *TableSpec
	OptCreatePartition *OptCreatePartition
	OptLike            *OptLike
	OptLocate          *OptLocate
	SchemaSpec         *SchemaSpec
	IndexSpec          *IndexSpec
	PartitionSpec      *PartitionSpec
	SequenceSpec       *SequenceSpec
	TriggerSpec        *TriggerSpec
	ProcedureSpec      *ProcedureSpec
	DDLType            ddlType
	ViewSpec           *ViewSpec
	ServerSpec         *ServerSpec
	SynonymSpec        *SynonymSpec
	ConstraintState    *ConstraintState
	ConstraintDef      *ConstraintDefinition
	ChangeSpec         *ChangeSpec
	ModifySpec         *ModifySpec
	PackageSpec        *PackageSpec
	PackageBody        *PackageBody
	PackageAlterBody   *PackageAlterBody
	FunctionBody       *FunctionBody
	FunctionAlterBody  *FunctionAlterBody
	ProcedureBody      *ProcedureBody
	ProcedureAlterBody *ProcedureAlterBody
	OptPartition       *OptPartition
}

// DDL strings.
const (
	CreateStr      = "create"
	AlterStr       = "alter"
	DropStr        = "drop"
	RenameStr      = "rename"
	AlterRenameStr = "alterRename"
	TruncateStr    = "truncate"
)

// Format formats the node.
func (node *DDL) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	// all branch of switch maybe used variables
	var notExists, exists string
	if node.IfNotExists {
		notExists = " if not exists"
	}
	if node.IfExists {
		exists = " if exists"
	}
	commentsBuf := NewTrackedBuffer(nil)
	if node.Comments != nil {
		node.Comments.Format(commentsBuf)
	}
	comments := " " + strings.TrimSpace(commentsBuf.String())
	if comments == " " {
		comments = ""
	}

	// format according to node.Action
	switch node.Action {
	case CreateStr:
		if node.OptLike != nil {
			buf.Myprintf("%s table%s %v %v", node.Action, notExists, node.NewName, node.OptLike)
		} else if node.IndexSpec != nil {
			buf.Myprintf("create %v", node.IndexSpec)
		} else if node.SequenceSpec != nil {
			buf.Myprintf("create sequence%s %v", notExists, node.NewName)
		} else if node.ViewSpec != nil {
			if node.IfNotExists {
				buf.Myprintf("create view if not exists %v", node.ViewSpec)
			} else {
				buf.Myprintf("create view %v", node.ViewSpec)
			}
		} else if node.SchemaSpec != nil {
			buf.Myprintf("create schema%s %v", notExists, node.SchemaSpec)
		} else if node.PackageSpec != nil {
			if node.CreateOrReplace {
				buf.Myprintf("CREATE OR REPLACE PACKAGE %v", node.PackageSpec)
			} else {
				buf.Myprintf("CREATE PACKAGE %v", node.PackageSpec)
			}
		} else if node.PackageBody != nil {
			if node.CreateOrReplace {
				buf.Myprintf("CREATE OR REPLACE PACKAGE BODY %v", node.PackageBody)
			} else {
				buf.Myprintf("CREATE PACKAGE BODY %v", node.PackageBody)
			}
		} else if node.FunctionBody != nil {
			if node.CreateOrReplace {
				buf.Myprintf("CREATE OR REPLACE %v", node.FunctionBody)
			} else {
				buf.Myprintf("CREATE %v", node.FunctionBody)
			}
		} else if node.ProcedureBody != nil {
			if node.CreateOrReplace {
				buf.Myprintf("CREATE OR REPLACE %v", node.ProcedureBody)
			} else {
				buf.Myprintf("CREATE %v", node.ProcedureBody)
			}
		} else if node.TableSpec == nil {
			buf.Myprintf("%s table%s%s %v", node.Action, comments, notExists, node.NewName)
		} else {
			buf.Myprintf("%s table%s %v %v", node.Action, notExists, node.NewName, node.TableSpec)
		}
	case DropStr:
		if node.IndexSpec != nil && node.IndexSpec.IdxDef != nil && node.IndexSpec.IdxDef.Info != nil {
			buf.Myprintf("drop index %v on %v", node.IndexSpec.IdxDef.Info.Name, node.NewName)
		} else if node.SchemaSpec != nil {
			buf.Myprintf("drop schema%s %v", exists, node.SchemaSpec.SchemaName)
		} else if node.ServerSpec != nil {
			buf.Myprintf("drop server%s %v", exists, node.ServerSpec.ServerName)
		} else if node.ProcedureSpec != nil {
			buf.Myprintf("%s procedure%s%s %v", node.Action, comments, exists, node.Table)
		} else if node.ViewSpec != nil {
			buf.Myprintf("%s view%s%s %v", node.Action, comments, exists, node.Table)
		} else if node.TriggerSpec != nil {
			buf.Myprintf("%s trigger%s%s %v", node.Action, comments, exists, node.Table)
		} else if node.SequenceSpec != nil {
			buf.Myprintf("%s sequence%s%s %v", node.Action, comments, exists, node.Table)
		} else if node.FunctionBody != nil {
			buf.Myprintf("%s FUNCTION%s %v;", strings.ToUpper(node.Action), exists, node.FunctionBody.FuncName)
		} else if node.ProcedureBody != nil {
			buf.Myprintf("%s PROCEDURE%s %v;", strings.ToUpper(node.Action), exists, node.ProcedureBody.ProcName)
		} else if node.PackageSpec != nil {
			buf.Myprintf("%s PACKAGE%s %v;", strings.ToUpper(node.Action), exists, node.PackageSpec.PackageName)
		} else if node.PackageBody != nil {
			buf.Myprintf("%s PACKAGE BODY%s %v;", strings.ToUpper(node.Action), exists, node.PackageBody.PackageName)
		} else {
			buf.Myprintf("%s table%s%s %v", node.Action, comments, exists, node.Table)
		}
	case RenameStr, AlterRenameStr:
		buf.Myprintf("%s table %v to %v", RenameStr, node.Table, node.NewName)
	case AlterStr:
		if node.PartitionSpec != nil {
			buf.Myprintf("%s table %v %v", node.Action, node.Table, node.PartitionSpec)
		} else if node.SchemaSpec != nil {
			buf.Myprintf("%s schema %v", node.Action, node.SchemaSpec.SchemaName)
		} else if node.ConstraintDef != nil {
			if _, ok := node.ConstraintDef.Detail.(*ForeignKeyDetail); ok {
				buf.Myprintf("%s table %v add %v", node.Action, node.Table, node.ConstraintDef)
			} else {
				buf.Myprintf("%s table %v", node.Action, node.Table)
			}
		} else if node.ChangeSpec != nil {
			buf.Myprintf("%s table %v %v", node.Action, node.Table, node.ChangeSpec)
		} else if node.ModifySpec != nil {
			buf.Myprintf("%s table %v %v", node.Action, node.Table, node.ModifySpec)
		} else if node.ViewSpec != nil {
			buf.Myprintf("%s view %v", node.Action, node.ViewSpec)
		} else if node.FunctionAlterBody != nil {
			buf.Myprintf("%s %v", strings.ToUpper(node.Action), node.FunctionAlterBody)
		} else if node.ProcedureAlterBody != nil {
			buf.Myprintf("%s %v", strings.ToUpper(node.Action), node.ProcedureAlterBody)
		} else if node.PackageAlterBody != nil {
			buf.Myprintf("%s %v", strings.ToUpper(node.Action), node.PackageAlterBody)
		} else {
			buf.Myprintf("%s table %v", node.Action, node.Table)
		}
	default:
		buf.Myprintf("%s table %v", node.Action, node.Table)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *DDL) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Table,
		node.NewName,
	)
}

func (node *DDL) PreCheckTable() error {
	if node.TableSpec != nil && node.OptCreatePartition != nil &&
		node.OptCreatePartition.OptUsing.String() == "hash" &&
		len(node.OptCreatePartition.Column) > 0 {
		column := node.OptCreatePartition.Column[0]
		sf := node.OptCreatePartition.ShardFunc.String()
		columnName := column.String()
		typ, err := node.TableSpec.GetColumnTypeByName(columnName)
		if err != nil {
			log.Warningf("failed to get column type, err is %s", err.Error())
			return err
		}
		if IsCharType(typ) && IsNumericFunc(sf) {
			log.Warningf("numeric function is not support for char type, column name is %s", columnName)
			return errors.New(errors.ErrNotSupport, "column type is not compatible with shard function")
		}
		if !IsCompatible(typ, sf) {
			log.Warningf("column type is not compatible with shard function, column name is %s", columnName)
			return errors.New(errors.ErrNotSupport, "column type is not compatible with shard function")
		}
	}
	return nil
}

// RetrieveIndexInfo retrieve the information about index
func (node *DDL) RetrieveIndexInfo() (indexName, tableName, database string, err error) {
	if node == nil || node.IndexSpec == nil {
		return "", "", "", errors.New(errors.ErrInvalidArgument, "index spec is empty")
	}
	indexName = strings.ToLower(node.IndexSpec.IdxDef.Info.Name.String())
	tableName = strings.ToLower(node.NewName.Name.String())
	database = strings.ToLower(node.NewName.Qualifier.String())
	return indexName, tableName, database, nil
}

// GetSequenceDDLSQL returns a sql to insert one row to the sequence metadata table
func (node *DDL) GetSequenceDDLSQL(insertData bool) (sql string, err error) {
	if node.SequenceSpec == nil {
		return "", errors.New(errors.ErrInvalidArgument, "node is not a sequence ddl statement")
	}

	switch node.Action {
	case CreateStr:
		{
			startIndex, err := strconv.Atoi(string(node.SequenceSpec.StartIndex.Val))
			if err != nil {
				return "", err
			}
			if startIndex < 0 {
				return "", errors.New(errors.ErrSeqStartValue, "sequence start index could not be negative")
			}
			cacheSize, err := strconv.Atoi(string(node.SequenceSpec.CacheSize.Val))
			if err != nil {
				return "", err
			}
			if cacheSize < 1 {
				return "", errors.New(errors.ErrSeqCache, "sequence cache size could not be less than 1")
			}

			if insertData {
				sql = "INSERT INTO `" + strings.ToLower(node.SequenceSpec.FullName.Name.String()) + "` (" +
					"`next_not_cached_value`, " +
					"`minimum_value`," +
					"`maximum_value`," +
					"`start_value`," +
					"`increment`," +
					"`cache_size`," +
					"`cycle_option`," +
					"`cycle_count`," +
					"`_kundb_dummy_next_val`," +
					"`_kundb_dummy_seq_name`)" +
					"VALUES (" +
					string(node.SequenceSpec.StartIndex.Val) + "," +
					string(node.SequenceSpec.StartIndex.Val) + "," +
					"9223372036854775806," +
					string(node.SequenceSpec.StartIndex.Val) + "," +
					"1," +
					string(node.SequenceSpec.CacheSize.Val) + "," +
					"0," +
					"0," +
					"1," +
					"'" + strings.ToLower(node.SequenceSpec.FullName.Name.String()) + "')"
				return sql, nil
			}
			sql = "CREATE TABLE `" + strings.ToLower(node.SequenceSpec.FullName.Name.String()) + "` (" +
				"`next_not_cached_value` bigint(21) NOT NULL DEFAULT 1," +
				"`minimum_value` bigint(21) NOT NULL DEFAULT 1," +
				"`maximum_value` bigint(21) NOT NULL DEFAULT 9223372036854775806," +
				"`start_value` bigint(21) NOT NULL DEFAULT 1 COMMENT 'start value when sequences is created or value if RESTART is used'," +
				"`increment` bigint(21) NOT NULL DEFAULT 1 COMMENT 'increment value'," +
				"`cache_size` bigint(21) unsigned NOT NULL DEFAULT 100," +
				"`cycle_option` tinyint(1) unsigned NOT NULL DEFAULT 0 COMMENT '0 if no cycles are allowed, 1 if the sequence should begin a new cycle when maximum_value is passed'," +
				"`cycle_count` bigint(21) NOT NULL DEFAULT 0 COMMENT 'How many cycles have been done'," +
				"`_kundb_dummy_next_val` int NOT NULL DEFAULT 1 COMMENT 'dummy column for kundb query plan generation'," +
				"`_kundb_dummy_seq_name` varchar(256) COMMENT 'dummy column for kundb query routing'," +
				"primary key (_kundb_dummy_seq_name)) comment 'vitess_sequence'"
			return sql, nil
		}
	case DropStr:
		{
			exists := ""
			if node.IfExists {
				exists = " if exists"
			}
			return "drop table" + exists + " `" + strings.ToLower(node.SequenceSpec.FullName.Name.String()) + "`", nil
		}
	}
	return "", fmt.Errorf("action %v not supported for sequence", node.Action)
}

// FormatNoRefPart returns a sql to DDL with partition option
func (node *DDL) FormatNoRefPart() (sql string, err error) {
	switch node.Action {
	case CreateStr, AlterStr:
		{
			if node.OptCreatePartition != nil && node.OptCreatePartition.OptUsing.Lowered() == ReferencePartition {
				tempConstraints := node.TableSpec.Constraints
				node.TableSpec.Constraints = nil
				sql := strings.ToLower(String(node))
				node.TableSpec.Constraints = tempConstraints
				return sql, nil
			}
			return strings.ToLower(String(node)), nil
		}
	case DropStr:
	}
	return "", fmt.Errorf("FormatNoRefPart: action %v not supported for table", node.Action)
}

// GetIndexDDLSQL returns a sql to DDL with partition option
// newSQL is routed to mysql, mfedSQL is routed to mfed
func (node *DDL) GetIndexDDLSQL(curDatabase string) (newSQL, mfedSQL string, err error) {
	switch node.Action {
	case CreateStr, DropStr:
		mfedSQL = String(node)
		first := true
		buff := NewTrackedBuffer(func(buf *TrackedBuffer, node SQLNode) {
			switch node := node.(type) {
			case TableName:
				tName := node.Name.String()
				if first {
					buf.Myprintf("%s", tName)
					first = false
				} else {
					dbName := node.Qualifier.String()
					if dbName == "" {
						dbName = curDatabase
					}
					foo := NewTableName(dbName, tName)
					buf.Myprintf("%s", String(foo))
				}
				return
			default:
				node.Format(buf)
				return
			}
		})
		node.Format(buff)
		newSQL = buff.String()
		return newSQL, mfedSQL, nil
	}
	return "", "", fmt.Errorf("GetIndexDDLSQL: action %v not supported for index", node.Action)
}

// DCL contains the information needed for privilege manipulation
type DCL struct {
	Action      string
	Privileges  Privileges
	GrantType   string
	GrantIdent  *GrantIdent
	Principals  Principals
	GrantOption bool
	GenPasswd   bool
	IfExists    bool // used for drop user
	Comments    Comments
}

// DCL strings
const (
	GrantStr      = "grant"
	RevokeStr     = "revoke"
	DropUserStr   = "drop user"
	CreateUserStr = "create user"
	DropRoleStr   = "drop role"
	CreateRoleStr = "create role"
)

// DCL strings
const (
	PrincipalUser = "user"
	PrincipalRole = "role"
)

// Grant Types
const (
	GrantTypeTable     = "table "
	GrantTypeFunction  = "function "
	GrantTypeProcedure = "procedure "
)

func (node *DCL) randStr() string {
	rand.Seed(time.Now().UnixNano())
	digits := "0123456789"
	specials := "~=+%^*/()[]{}/!@#$?|"
	all := "ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		digits + specials
	length := 20
	buf := make([]byte, length)
	for i := 0; i < length; i++ {
		buf[i] = all[rand.Intn(len(all))]
	}
	return string(buf)
}

// FormatCreateUserOrRole formats the create user/role sql
func (node *DCL) FormatCreateUserOrRole(buf *TrackedBuffer) {
	if node == nil {
		return
	}

	if node.Action != CreateUserStr && node.Action != CreateRoleStr {
		panic("not a create user/role statement")
	}

	ifExists := ""
	if node.IfExists {
		ifExists = " if not exists"
	}
	buf.Myprintf("%s%s ", node.Action, ifExists)

	if node.GenPasswd {
		var prefix string
		for _, u := range node.Principals.PrincipalDetail {
			buf.Myprintf("%s%v identified by ''", prefix, u)
			prefix = ", "
		}
	} else {
		buf.Myprintf("%v", node.Principals)
	}
}

// Format formats the node into a buffer
func (node *DCL) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}

	if node.Action == CreateUserStr || node.Action == CreateRoleStr {
		node.FormatCreateUserOrRole(buf)
		return
	}

	if node.Action == DropUserStr || node.Action == DropRoleStr {
		ifExists := ""
		if node.IfExists {
			ifExists = " if exists"
		}
		buf.Myprintf("%s%s %v", node.Action, ifExists, node.Principals)
		return
	}

	var prep string
	if node.Action == GrantStr {
		prep = "to"
	} else {
		prep = "from"
	}
	if node.GrantType == PrincipalRole {
		// Grant/revoke role to/from user
		buf.Myprintf("%s %v %s %v", node.Action, node.Privileges, prep, node.Principals)
	} else {
		buf.Myprintf("%s %v on %s%v %s %v", node.Action, node.Privileges, node.GrantType, node.GrantIdent, prep, node.Principals)
	}

	if node.Action == GrantStr {
		if node.GrantOption {
			if node.GrantType == PrincipalRole {
				buf.WriteString(" with admin option")
			} else {
				buf.WriteString(" with grant option")
			}
		}
	}
}

// WalkSubtree walks the subtree rooted at this node
func (node *DCL) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Privileges,
		node.GrantIdent,
		node.Principals,
	)
}

// GrantIdent is the target object the DCL acts on
type GrantIdent struct {
	// VitessDbPrefix is the prefix added by vitess to the MySQL database
	// corresponding to the respective keyspace, such as "vt_"
	VitessDbPrefix       string
	WildcardKeyspaceName bool
	WildcardTableName    bool
	Schema               TableIdent
	Table                TableIdent
}

// Format formats the node into a buffer
func (node *GrantIdent) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	if !node.Schema.IsEmpty() {
		if node.WildcardTableName {
			buf.Myprintf("%s%v.*", node.VitessDbPrefix, node.Schema)
			return
		}
		buf.Myprintf("%s%v.%v", node.VitessDbPrefix, node.Schema, node.Table)
	} else if node.WildcardKeyspaceName {
		if node.WildcardTableName {
			buf.Myprintf("*.*")
		} else {
			buf.Myprintf("*.%v", node.Table)
		}
	} else if !node.Table.IsEmpty() {
		if node.WildcardTableName {
			buf.Myprintf("*")
		} else {
			buf.Myprintf("%v", node.Table)
		}
	}
}

// WalkSubtree walks the subtree rooted at this node
func (node *GrantIdent) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Schema,
		node.Table,
	)
}

// Principals is a list of Principal objects
type Principals struct {
	PrincipalType   string
	PrincipalDetail PrincipalDetail
}

// Format formats the node into a buffer
func (node Principals) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v", node.PrincipalDetail)
}

// WalkSubtree walks the nodes of the subtree.
func (node Principals) WalkSubtree(visit Visit) error {
	if err := Walk(visit, node.PrincipalDetail); err != nil {
		return err
	}
	return nil
}

// PrincipalDetail is the container for users or roles
type PrincipalDetail []*Principal

// Format formats the node into a buffer
func (node PrincipalDetail) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		if n.t == "" {
			buf.Myprintf("%s%s", prefix, n.v)
		} else {
			buf.Myprintf("%s%v", prefix, n)
		}
		prefix = ", "
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node PrincipalDetail) WalkSubtree(visit Visit) error {
	for _, n := range node {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}
	return nil
}

// Principal is wrapping of user or role
type Principal struct {
	// Principal name stored in v, and type in t
	v, t string
}

// NewUser creates a new user
func NewUser(str string) *Principal {
	return &Principal{v: strings.ToLower(str), t: PrincipalUser}
}

func (node Principal) SetNullUser(str string) *Principal {
	return &Principal{v: strings.ToLower(str), t: ""}
}

// NewRole creates a new role
func NewRole(str string) *Principal {
	return &Principal{v: strings.ToLower(str), t: PrincipalRole}
}

// Name gets the principal name
func (node Principal) Name() string {
	return node.v
}

// Format formats the node into a buffer
func (node Principal) Format(buf *TrackedBuffer) {
	if node.t == PrincipalUser {
		buf.Myprintf("'%s'@'%s'", node.v, "localhost")
	} else {
		buf.Myprintf("'%s'", node.v)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node Principal) WalkSubtree(visit Visit) error {
	return nil
}

// Privilege is the privilege type
type Privilege struct {
	Privilege string
	Columns   Columns
}

// Format formats the node into a buffer
func (node *Privilege) Format(buf *TrackedBuffer) {
	if node.Columns == nil {
		buf.Myprintf("%s", node.Privilege)
	} else {
		buf.Myprintf("%s (", node.Privilege)
		for i, col := range node.Columns {
			if i == 0 {
				buf.Myprintf("%v", col)
			} else {
				buf.Myprintf(", %v", col)

			}
		}
		buf.Myprintf(")")
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *Privilege) WalkSubtree(visit Visit) error {
	return nil
}

// Privileges represents a list of Privilege
type Privileges []*Privilege

// Format formats the node.
func (node Privileges) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Myprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node Privileges) WalkSubtree(visit Visit) error {
	for _, n := range node {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}
	return nil
}

// ViewSpec describe the view properties
type ViewSpec struct {
	ViewName        TableName
	Definition      string
	IsReplace       bool
	IsScatter       bool
	OnShards        bool
	Select          SelectStatement
	WithCheckOption string
}

// Format formats the node.
func (node *ViewSpec) Format(buf *TrackedBuffer) {
	if node.WithCheckOption == "" {
		buf.Myprintf("%v as %v", node.ViewName, node.Select)
	} else {
		buf.Myprintf("%v as %v %s", node.ViewName, node.Select, node.WithCheckOption)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *ViewSpec) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	Walk(visit, node.ViewName, node.Select)

	return nil
}

// SequenceSpec describe the sequence initialization options
// create sequence seq_name [start with X cache Y]
type SequenceSpec struct {
	FullName   TableName
	StartIndex *SQLVal
	CacheSize  *SQLVal
	Options    TableOptions
}

// TriggerSpec describe the trigger definition
type TriggerSpec struct {
	TriggerTime   string
	TriggerEvent  string
	TriggerTarget TableName
	TriggerName   TableName
	OnShards      bool
}

type DataType struct {
	NativeType      *ColumnType
	OptWith         string
	OptLocal        string
	OptCharSet      IdExpressionList
	OptIntervalFrom *PlsqlExpression
	FromTimePeriod  string
	OptIntervalTo   *PlsqlExpression
	ToTimePeriod    string
}

func (node *DataType) Format(buf *TrackedBuffer) {
	if node.NativeType != nil {
		buf.Myprintf("%v", node.NativeType)
		if node.OptWith != "" {
			buf.Myprintf(" WITH")
			if node.OptLocal != "" {
				buf.Myprintf(" LOCAL")
			}
			buf.Myprintf(" TIME ZONE")
		} else if node.OptCharSet != nil {
			buf.Myprintf(" CHARACTER SET %v", node.OptCharSet)
		}
	} else {
		buf.Myprintf("INTERVAL %s", node.FromTimePeriod)
		if node.OptIntervalFrom != nil {
			buf.Myprintf(" (%v)", node.OptIntervalFrom)
		}
		buf.Myprintf(" TO %s", node.ToTimePeriod)
		if node.OptIntervalTo != nil {
			buf.Myprintf(" (%v)", node.OptIntervalTo)
		}
	}
}

func (node *DataType) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.NativeType,
		node.OptCharSet,
		node.OptIntervalFrom,
		node.OptIntervalTo,
	)
}

type TypeSpec struct {
	DataType    *DataType
	RefOpt      string
	TypeName    string
	TypeNameOpt string
}

func (node *TypeSpec) Format(buf *TrackedBuffer) {
	if node.DataType != nil {
		buf.Myprintf("%v", node.DataType)
	} else if node.TypeName != "" {
		if node.RefOpt != "" {
			if node.TypeNameOpt != "" {
				buf.Myprintf("REF %s%s", node.TypeName, node.TypeNameOpt)
			} else {
				buf.Myprintf("REF %s", node.TypeName)
			}
		} else {
			if node.TypeNameOpt != "" {
				buf.Myprintf("%s%s", node.TypeName, node.TypeNameOpt)
			} else {
				buf.Myprintf("%s", node.TypeName)
			}
		}
	}
}

func (node *TypeSpec) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.DataType,
	)
}

type Parameter struct {
	ParameterName string
	ModeOpt       string
	TypeOpt       *TypeSpec
}

func (node *Parameter) Format(buf *TrackedBuffer) {
	if node.ModeOpt != "" {
		if node.TypeOpt != nil {
			buf.Myprintf("%s %s %v", node.ParameterName, node.ModeOpt, node.TypeOpt)
		} else {
			buf.Myprintf("%s %s", node.ParameterName, node.ModeOpt)
		}
	} else {
		if node.TypeOpt != nil {
			buf.Myprintf("%s %v", node.ParameterName, node.TypeOpt)
		} else {
			buf.Myprintf("%s", node.ParameterName)
		}
	}
}

func (node *Parameter) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.TypeOpt,
	)
}

type ParameterList []*Parameter

func (node ParameterList) Format(buf *TrackedBuffer) {
	for index, parameter := range node {
		if index == len(node)-1 {
			buf.Myprintf("%v", parameter)
		} else {
			buf.Myprintf("%v, ", parameter)
		}
	}
}

func (node ParameterList) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, parameter := range node {
		if err := Walk(visit, parameter); err != nil {
			return err
		}
	}
	return nil
}

func (*FunctionSpec) PLDeclaration() {}
func (*FunctionSpec) PLPackageBody() {}

// FunctionSpec describe the function definition
// create sharded function func_name func_def
type FunctionSpec struct {
	FullName      TableName
	OnShards      bool
	ParametersOpt ParameterList
	ReturnType    *TypeSpec
	ReturnOpt     FuncReturnSuffixList
}

// Format formats the node.
func (node *FunctionSpec) Format(buf *TrackedBuffer) {
	buf.Myprintf("FUNCTION %v (", node.FullName)
	buf.Myprintf("%v", node.ParametersOpt)
	buf.Myprintf(")")
	buf.Myprintf(" RETURN %v", node.ReturnType)
	if node.ReturnOpt != nil {
		buf.Myprintf(" %v", node.ReturnOpt)
	}
}

func (node *FunctionSpec) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.FullName, node.ParametersOpt, node.ReturnType); err != nil {
		return err
	}
	return nil
}

func (*ProcedureSpec) PLDeclaration() {}
func (*ProcedureSpec) PLPackageBody() {}

// `Procedure`Spec describe the procedure definition
type ProcedureSpec struct {
	ProcName      TableName
	ParametersOpt ParameterList
}

func (node *ProcedureSpec) AddParameter(parameter *Parameter) {
	node.ParametersOpt = append(node.ParametersOpt, parameter)
}

// Format formats the node.
func (node *ProcedureSpec) Format(buf *TrackedBuffer) {
	buf.Myprintf("PROCEDURE %v", node.ProcName)
	if node.ParametersOpt != nil {
		buf.Myprintf(" (%v)", node.ParametersOpt)
	}
}

func (node *ProcedureSpec) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.ProcName, node.ParametersOpt); err != nil {
		return err
	}
	return nil
}

func (*VariableDeclaration) PLDeclaration() {}
func (*VariableDeclaration) PLPackageBody() {}

type DefaultValuePart struct {
	AssignExpression  *PlsqlExpression
	DefaultExpression *PlsqlExpression
}

// Format formats the node.
func (node *DefaultValuePart) Format(buf *TrackedBuffer) {
	if node.AssignExpression != nil {
		buf.Myprintf(":= %v", node.AssignExpression)
	} else if node.DefaultExpression != nil {
		buf.Myprintf("DEFAULT %v", node.DefaultExpression)
	}
}

func (node *DefaultValuePart) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.AssignExpression, node.DefaultExpression); err != nil {
		return err
	}
	return nil
}

type VariableDeclaration struct {
	VariableName        string
	ConstraintOpt       string
	TypeSpec            *TypeSpec
	NullOpt             string
	DefaultValuePartOpt *DefaultValuePart
}

// Format formats the node.
func (node *VariableDeclaration) Format(buf *TrackedBuffer) {
	buf.Myprintf("%s", node.VariableName)
	if node.ConstraintOpt != "" {
		buf.Myprintf(" %s", node.ConstraintOpt)
	}
	buf.Myprintf(" %v", node.TypeSpec)
	if node.NullOpt != "" {
		buf.Myprintf(" %s", node.NullOpt)
	}
	if node.DefaultValuePartOpt != nil {
		buf.Myprintf(" %v", node.DefaultValuePartOpt)
	}
}

func (node *VariableDeclaration) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.TypeSpec, node.DefaultValuePartOpt); err != nil {
		return err
	}
	return nil
}

type ParameterSpecType struct {
	ModeOpt  string
	TypeSpec *TypeSpec
}

func (node *ParameterSpecType) Format(buf *TrackedBuffer) {
	if node.ModeOpt != "" {
		buf.Myprintf("%s %v", node.ModeOpt, node.TypeSpec)
	} else {
		buf.Myprintf("%v", node.TypeSpec)
	}
}

func (node *ParameterSpecType) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.TypeSpec); err != nil {
		return err
	}
	return nil
}

type ParameterSpec struct {
	ParameterName   string
	TypeOpt         *ParameterSpecType
	DefaultValueOpt *DefaultValuePart
}

func (node *ParameterSpec) Format(buf *TrackedBuffer) {
	buf.Myprintf("%s", node.ParameterName)
	if node.TypeOpt != nil {
		buf.Myprintf(" %v", node.TypeOpt)
	}
	if node.DefaultValueOpt != nil {
		buf.Myprintf(" %v", node.DefaultValueOpt)
	}
}

func (node *ParameterSpec) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.TypeOpt,
	)
}

type ParameterSpecList []*ParameterSpec

func (node ParameterSpecList) Format(buf *TrackedBuffer) {
	for index, parameterSpec := range node {
		if index == len(node)-1 {
			buf.Myprintf("%v", parameterSpec)
		} else {
			buf.Myprintf("%v, ", parameterSpec)
		}
	}
}

func (node ParameterSpecList) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, parameterSpec := range node {
		if err := Walk(visit, parameterSpec); err != nil {
			return err
		}
	}
	return nil
}

func (*CursorDeclaration) PLDeclaration() {}
func (*CursorDeclaration) PLPackageBody() {}

type CursorDeclaration struct {
	CursorName        string
	ParameterSpecsOpt ParameterSpecList
	ReturnTypeOpt     *TypeSpec
	IsStatementOpt    SelectStatement
}

// Format formats the node.
func (node *CursorDeclaration) Format(buf *TrackedBuffer) {
	buf.Myprintf("CURSOR %s (", node.CursorName)
	if node.ParameterSpecsOpt != nil {
		buf.Myprintf("%v) ", node.ParameterSpecsOpt)
	}
	if node.ReturnTypeOpt != nil {
		buf.Myprintf("RETURN %v ", node.ReturnTypeOpt)
	}
	if node.IsStatementOpt != nil {
		buf.Myprintf("IS %v", node.IsStatementOpt)
	}
}

func (node *CursorDeclaration) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.ParameterSpecsOpt, node.ReturnTypeOpt, node.IsStatementOpt); err != nil {
		return err
	}
	return nil
}

func (*ExceptionDeclaration) PLDeclaration() {}
func (*ExceptionDeclaration) PLPackageBody() {}

type ExceptionDeclaration struct {
	ExceptionName string
}

func (node *ExceptionDeclaration) Format(buf *TrackedBuffer) {
	buf.Myprintf("%s EXCEPTION", node.ExceptionName)
}

func (node *ExceptionDeclaration) WalkSubtree(visit Visit) error {
	return nil
}

type LogicalExpression struct {
	VitessExpression  Expr
	CursorPercentName TableName
	CursorPercentOpt  string
}

func (node *LogicalExpression) Format(buf *TrackedBuffer) {
	if node.VitessExpression != nil {
		buf.Myprintf("%v", node.VitessExpression)
	} else if !node.CursorPercentName.IsEmpty() {
		buf.Myprintf("%v %s", node.CursorPercentName, strings.ToUpper(node.CursorPercentOpt))
	}
}

func (node *LogicalExpression) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.VitessExpression, node.CursorPercentName); err != nil {
		return err
	}
	return nil
}

type CursorExpression struct {
	SubQuery *Subquery
}

func (node *CursorExpression) Format(buf *TrackedBuffer) {
	buf.Myprintf("CURSOR %v", node.SubQuery)
}

func (node *CursorExpression) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.SubQuery); err != nil {
		return err
	}
	return nil
}

type PlsqlExpressions []*PlsqlExpression

func (node PlsqlExpressions) Format(buf *TrackedBuffer) {
	for i, expr := range node {
		if i == 0 {
			buf.Myprintf("%v", expr)
		} else {
			buf.Myprintf(", %v", expr)
		}
	}
}

func (node PlsqlExpressions) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, expr := range node {
		if err := Walk(visit, expr); err != nil {
			return err
		}
	}
	return nil
}

type PlsqlExpression struct {
	CursorExpre  *CursorExpression
	LogicalExpre *LogicalExpression
}

func (node *PlsqlExpression) Format(buf *TrackedBuffer) {
	if node.CursorExpre != nil {
		buf.Myprintf("%v", node.CursorExpre)
	}
	if node.LogicalExpre != nil {
		buf.Myprintf("%v", node.LogicalExpre)
	}
}

func (node *PlsqlExpression) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.CursorExpre, node.CursorExpre); err != nil {
		return err
	}
	return nil
}

type SubTypeRange struct {
	LowValue  Expr
	HighValue Expr
}

func (node *SubTypeRange) Format(buf *TrackedBuffer) {
	buf.Myprintf("RANGE %v .. %v", node.LowValue, node.HighValue)
}

func (node *SubTypeRange) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.LowValue, node.LowValue); err != nil {
		return err
	}
	return nil
}

func (*SubTypeDeclaration) PLDeclaration() {}
func (*SubTypeDeclaration) PLPackageBody() {}

type SubTypeDeclaration struct {
	SubTypeName string
	Type        *TypeSpec
	RangeOpt    *SubTypeRange
	NullOpt     string
}

func (node *SubTypeDeclaration) Format(buf *TrackedBuffer) {
	buf.Myprintf("SUBTYPE %s IS %v ", node.SubTypeName, node.Type)
	if node.RangeOpt != nil {
		buf.Myprintf("%v", node.RangeOpt)
	}
	if node.NullOpt != "" {
		buf.Myprintf(" %s", node.NullOpt)
	}
}

func (node *SubTypeDeclaration) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.Type, node.RangeOpt); err != nil {
		return err
	}
	return nil
}

type CursorTypeDef struct {
	ReturnTypeOpt *TypeSpec
}

func (node *CursorTypeDef) Format(buf *TrackedBuffer) {
	buf.Myprintf("REF CURSOR")
	if node.ReturnTypeOpt != nil {
		buf.Myprintf(" RETURN %v", node.ReturnTypeOpt)
	}
}

func (node *CursorTypeDef) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.ReturnTypeOpt); err != nil {
		return err
	}
	return nil
}

type FieldSpec struct {
	columnName      TableName
	TypeOpt         *TypeSpec
	NotNullOpt      string
	DefaultValueOpt *DefaultValuePart
}

func (node *FieldSpec) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v", node.columnName)
	if node.TypeOpt != nil {
		buf.Myprintf(" %v", node.TypeOpt)
	}
	if node.NotNullOpt != "" {
		buf.Myprintf(" %s", node.NotNullOpt)
	}
	if node.DefaultValueOpt != nil {
		buf.Myprintf(" %v", node.DefaultValueOpt)
	}
}

func (node *FieldSpec) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.columnName, node.TypeOpt); err != nil {
		return err
	}
	return nil
}

type FieldSpecList []*FieldSpec

func (node FieldSpecList) Format(buf *TrackedBuffer) {
	for i, field := range node {
		if i == 0 {
			buf.Myprintf("%v", field)
		} else {
			buf.Myprintf(", %v", field)
		}
	}
}

func (node FieldSpecList) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, field := range node {
		if err := Walk(visit, field); err != nil {
			return err
		}
	}
	return nil
}

type RecordTypeDef struct {
	FieldSpecs FieldSpecList
}

func (node *RecordTypeDef) Format(buf *TrackedBuffer) {
	buf.Myprintf("RECORD (%v)", node.FieldSpecs)
}

func (node *RecordTypeDef) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.FieldSpecs); err != nil {
		return err
	}
	return nil
}

type VarrayTypeDef struct {
	VarrayPrefix string
	Expression   *PlsqlExpression
	Type         *TypeSpec
	NotNullOpt   string
}

func (node *VarrayTypeDef) Format(buf *TrackedBuffer) {
	buf.Myprintf("%s (%v) OF %v", node.VarrayPrefix, node.Expression, node.Type)
	if node.NotNullOpt != "" {
		buf.Myprintf(" %s", node.NotNullOpt)
	}
}

func (node *VarrayTypeDef) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.Expression, node.Type); err != nil {
		return err
	}
	return nil
}

type TableIndex struct {
	// todo
}

func (node *TableIndex) Format(buf *TrackedBuffer) {
	// todo
}

func (node *TableIndex) WalkSubtree(visit Visit) error {
	return nil
}

type TableTypeDef struct {
	Type          *TypeSpec
	TableIndexOpt *TableIndex
	NotNullOpt    string
}

func (node *TableTypeDef) Format(buf *TrackedBuffer) {
	buf.Myprintf("TABLE OF %v", node.Type)
	if node.TableIndexOpt != nil {
		buf.Myprintf(" %v", node.TableIndexOpt)
	}
	if node.NotNullOpt != "" {
		buf.Myprintf(" %s", node.NotNullOpt)
	}
}

func (node *TableTypeDef) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.Type, node.TableIndexOpt); err != nil {
		return err
	}
	return nil
}

func (*TypeDeclaration) PLDeclaration() {}
func (*TypeDeclaration) PLPackageBody() {}

type TypeDeclaration struct {
	TypeName      string
	TableTypeOpt  *TableTypeDef
	VarrayTypeOpt *VarrayTypeDef
	RecordTypeOpt *RecordTypeDef
	CursorTypeOpt *CursorTypeDef
}

func (node *TypeDeclaration) Format(buf *TrackedBuffer) {
	buf.Myprintf("TYPE %s IS", node.TypeName)
	if node.TableTypeOpt != nil {
		buf.Myprintf(" %v", node.TableTypeOpt)
	} else if node.VarrayTypeOpt != nil {
		buf.Myprintf(" %v", node.VarrayTypeOpt)
	} else if node.RecordTypeOpt != nil {
		buf.Myprintf(" %v", node.RecordTypeOpt)
	} else if node.CursorTypeOpt != nil {
		buf.Myprintf(" %v", node.CursorTypeOpt)
	}
}

func (node *TypeDeclaration) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.TableTypeOpt, node.VarrayTypeOpt, node.VarrayTypeOpt, node.RecordTypeOpt, node.CursorTypeOpt); err != nil {
		return err
	}
	return nil
}

type ExceptionHandler struct {
	ExceptionName  string
	SeqOfStatement *SeqOfStatement
}

func (node *ExceptionHandler) Format(buf *TrackedBuffer) {
	buf.Myprintf("EXCEPTION WHEN %s THEN %v", node.ExceptionName, node.SeqOfStatement)
}

func (node *ExceptionHandler) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.SeqOfStatement); err != nil {
		return err
	}
	return nil
}

type ExceptionHandlerList []*ExceptionHandler

func (node ExceptionHandlerList) Format(buf *TrackedBuffer) {
	for _, exception := range node {
		buf.Myprintf(" %v", exception)
	}
}

func (node ExceptionHandlerList) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, exception := range node {
		if err := Walk(visit, exception); err != nil {
			return err
		}
	}
	return nil
}

type PLStatement interface {
	PLStatement()
	SQLNode
}

type IdExpression struct {
	RegularIdOpt   string
	DelimitedIdOpt string
}

func (node *IdExpression) Format(buf *TrackedBuffer) {
	if node.RegularIdOpt != "" {
		buf.Myprintf("%s", node.RegularIdOpt)
	} else if node.DelimitedIdOpt != "" {
		buf.Myprintf("%s", node.DelimitedIdOpt)
	}
}

func (node *IdExpression) WalkSubtree(visit Visit) error {
	return nil
}

func (node *IdExpression) String() string {
	if node.RegularIdOpt != "" {
		return node.RegularIdOpt
	} else {
		return node.DelimitedIdOpt
	}
}

type IdExpressionList []*IdExpression

func (node IdExpressionList) Format(buf *TrackedBuffer) {
	for i, idExpression := range node {
		if i == 0 {
			buf.Myprintf("%v", idExpression)
		} else {
			buf.Myprintf(".%v", idExpression)
		}
	}
}

func (node IdExpressionList) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, idExpression := range node {
		if err := Walk(visit, idExpression); err != nil {
			return err
		}
	}
	return nil
}

type BindVariable struct {
	BindVarPrefix          string
	IndicatorOpt           string
	GeneralElementPartList GeneralElementParList
}

func (node *BindVariable) Format(buf *TrackedBuffer) {
	buf.Myprintf("%s", node.BindVarPrefix)
	if node.IndicatorOpt != "" {
		buf.Myprintf(" %s", node.IndicatorOpt)
	}
	if node.GeneralElementPartList != nil {
		buf.Myprintf("%v", node.GeneralElementPartList)
	}
}

func (node *BindVariable) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.GeneralElementPartList); err != nil {
		return err
	}
	return nil
}

type GeneralElementParList []*GeneralElementPart

func (node GeneralElementParList) Format(buf *TrackedBuffer) {
	var prefix string
	for _, ele := range node {
		prefix = "."
		buf.Myprintf("%s%v", prefix, ele)
	}
}

func (node GeneralElementParList) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, ele := range node {
		if err := Walk(visit, ele); err != nil {
			return err
		}
	}
	return nil
}

type GeneralElementPart struct {
	CharSetNameOpt  IdExpressionList
	IdExpressions   IdExpressionList
	LinkNameOpt     string
	FuncArgumentOpt SelectExprs
}

func (node *GeneralElementPart) Format(buf *TrackedBuffer) {
	if node.CharSetNameOpt != nil {
		buf.Myprintf("INTRODUCER %v ", node.CharSetNameOpt)
	}
	buf.Myprintf("%v", node.IdExpressions)
	if node.LinkNameOpt != "" {
		buf.Myprintf(" %s", node.LinkNameOpt)
	}
	if node.FuncArgumentOpt != nil {
		buf.Myprintf(" %v", node.FuncArgumentOpt)
	}
}

func (node *GeneralElementPart) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.CharSetNameOpt, node.IdExpressions, node.FuncArgumentOpt); err != nil {
		return err
	}
	return nil
}

type GeneralElement []*GeneralElementPart

func (node GeneralElement) Format(buf *TrackedBuffer) {
	var prefix string
	for _, ele := range node {
		buf.Myprintf("%s%v", prefix, ele)
		prefix = "."
	}
}

func (node GeneralElement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, ele := range node {
		if err := Walk(visit, ele); err != nil {
			return err
		}
	}
	return nil
}

func (*AssignmentStatement) PLStatement() {}

type AssignmentStatement struct {
	GeneralElementOpt *Identifier
	BindVariableOpt   *BindVariable
	Expression        Expr
}

func (node *AssignmentStatement) Format(buf *TrackedBuffer) {
	if node.GeneralElementOpt != nil {
		buf.Myprintf("%v", node.GeneralElementOpt)
	} else if node.BindVariableOpt != nil {
		buf.Myprintf("%v", node.BindVariableOpt)
	}
	buf.Myprintf(":= %v;", node.Expression)
}

func (node *AssignmentStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.GeneralElementOpt, node.BindVariableOpt, node.Expression); err != nil {
		return err
	}
	return nil
}

func (*ContinueStatement) PLStatement() {}

type ContinueStatement struct {
	LableNameOpt string
	ConditionOpt Expr
}

func (node *ContinueStatement) Format(buf *TrackedBuffer) {
	buf.Myprintf("CONTINUE")
	if node.LableNameOpt != "" {
		buf.Myprintf(" %s", node.LableNameOpt)
	}
	if node.ConditionOpt != nil {
		buf.Myprintf(" WHEN %v", node.ConditionOpt)
	}
	buf.Myprintf(";")
}

func (node *ContinueStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.ConditionOpt); err != nil {
		return err
	}
	return nil
}

func (*ExitStatement) PLStatement() {}

type ExitStatement struct {
	LableNameOpt string
	ConditionOpt Expr
}

func (node *ExitStatement) Format(buf *TrackedBuffer) {
	buf.Myprintf("EXIT")
	if node.LableNameOpt != "" {
		buf.Myprintf(" %s", node.LableNameOpt)
	}
	if node.ConditionOpt != nil {
		buf.Myprintf(" WHEN %v", node.ConditionOpt)
	}
	buf.Myprintf(";")
}

func (node *ExitStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.ConditionOpt); err != nil {
		return err
	}
	return nil
}

func (*GoToStatement) PLStatement() {}

type GoToStatement struct {
	LableName string
}

func (node *GoToStatement) Format(buf *TrackedBuffer) {
	buf.Myprintf("GOTO %s;", node.LableName)
}

func (node *GoToStatement) WalkSubtree(visit Visit) error {
	return nil
}

type ElseIf struct {
	Condition      Expr
	SeqOfStatement *SeqOfStatement
}

func (node *ElseIf) Format(buf *TrackedBuffer) {
	buf.Myprintf("ELSIF %v THEN %v", node.Condition, node.SeqOfStatement)
}

func (node *ElseIf) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.Condition, node.SeqOfStatement); err != nil {
		return err
	}
	return nil
}

type ElseIfList []*ElseIf

func (node ElseIfList) Format(buf *TrackedBuffer) {
	for i, elseif := range node {
		if i == 0 {
			buf.Myprintf("%v", elseif)
		} else {
			buf.Myprintf(" %v", elseif)
		}
	}
}

func (node ElseIfList) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, elseif := range node {
		if err := Walk(visit, elseif); err != nil {
			return err
		}
	}

	return nil
}

type ElsePart struct {
	SeqOfStatement *SeqOfStatement
}

func (node *ElsePart) Format(buf *TrackedBuffer) {
	buf.Myprintf("ELSE %v", node.SeqOfStatement)
}

func (node *ElsePart) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.SeqOfStatement); err != nil {
		return err
	}
	return nil
}

func (*IfStatement) PLStatement() {}

type IfStatement struct {
	Condition      Expr
	SeqOfStatement *SeqOfStatement
	ElseIfsPart    ElseIfList
	ElsePartOpt    *ElsePart
}

func (node *IfStatement) Format(buf *TrackedBuffer) {
	buf.Myprintf("IF %v THEN %v", node.Condition, node.SeqOfStatement)
	if node.ElseIfsPart != nil {
		buf.Myprintf(" %v", node.ElseIfsPart)
	}
	if node.ElsePartOpt != nil {
		buf.Myprintf(" %v", node.ElsePartOpt)
	}
	buf.Myprintf(" END IF;")
}

func (node *IfStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.Condition, node.SeqOfStatement, node.ElseIfsPart, node.ElsePartOpt); err != nil {
		return err
	}
	return nil
}

type LoopParam interface {
	LoopParam()
	SQLNode
}

func (*IndexLoopParam) LoopParam() {}

type IndexLoopParam struct {
	IndexName  *ColName
	ReverseOpt string
	LowBound   Expr
	upperBound Expr
}

func (node *IndexLoopParam) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v IN", node.IndexName)
	if node.ReverseOpt != "" {
		buf.Myprintf(" %s", node.ReverseOpt)
	}
	buf.Myprintf(" %v .. %v", node.LowBound, node.upperBound)
}

func (node *IndexLoopParam) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.IndexName, node.LowBound, node.upperBound); err != nil {
		return err
	}
	return nil
}

func (*RecordLoopParam) LoopParam() {}

type RecordLoopParam struct {
	RecordName          *ColName
	CursorNameOpt       *ColName
	PlsqlExpressionsOpt PlsqlExpressions
	SelectStatementOpt  *Select
}

func (node *RecordLoopParam) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v IN (", node.RecordName)
	if node.CursorNameOpt != nil {
		buf.Myprintf("%v", node.CursorNameOpt)
		if node.PlsqlExpressionsOpt != nil {
			buf.Myprintf("(%v)", node.PlsqlExpressionsOpt)
		}
	} else if node.SelectStatementOpt != nil {
		buf.Myprintf("%v", node.SelectStatementOpt)
	}
	buf.Myprintf(")")
}

func (node *RecordLoopParam) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.RecordName, node.CursorNameOpt, node.PlsqlExpressionsOpt, node.SelectStatementOpt); err != nil {
		return err
	}
	return nil
}

func (*LoopStatement) PLStatement() {}

type LoopStatement struct {
	WhileConditionOpt Expr
	ForLoopParamOpt   LoopParam
	SeqOfStatement    *SeqOfStatement
	EndLabelNameOpt   string
}

func (node *LoopStatement) Format(buf *TrackedBuffer) {
	if node.WhileConditionOpt != nil {
		buf.Myprintf("WHILE %v ", node.WhileConditionOpt)
	} else if node.ForLoopParamOpt != nil {
		buf.Myprintf("FOR %v ", node.ForLoopParamOpt)
	}
	buf.Myprintf("LOOP %v END LOOP", node.SeqOfStatement)
	if node.EndLabelNameOpt != "" {
		buf.Myprintf(" %s", node.EndLabelNameOpt)
	}
	buf.Myprintf(";")
}

func (node *LoopStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.WhileConditionOpt, node.ForLoopParamOpt, node.SeqOfStatement); err != nil {
		return err
	}
	return nil
}

type IndexName struct {
	Identifier      *Identifier
	IdExpressionOpt *IdExpression
}

func (node *IndexName) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v", node.Identifier)
	if node.IdExpressionOpt != nil {
		buf.Myprintf(".%v", node.IdExpressionOpt)
	}
}

func (node *IndexName) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.Identifier, node.IdExpressionOpt); err != nil {
		return err
	}
	return nil
}

type LowBound struct {
	// todo
}

type UpperBound struct {
	// todo
}

type BetweenBound struct {
	LowBound   Expr
	UpperBound Expr
}

func (node *BetweenBound) Format(buf *TrackedBuffer) {
	buf.Myprintf("BETWEEN %v AND %v", node.LowBound, node.UpperBound)
}

func (node *BetweenBound) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.LowBound, node.UpperBound); err != nil {
		return err
	}
	return nil
}

type BoundsClause struct {
	LowBound        Expr
	UpperBound      Expr
	IndicesOfOpt    *IndexName
	BetweenBoundOpt *BetweenBound
	ValuesOf        *IndexName
}

func (node *BoundsClause) Format(buf *TrackedBuffer) {
	if node.LowBound != nil {
		buf.Myprintf("%v .. %v", node.LowBound, node.UpperBound)
	} else if node.IndicesOfOpt != nil {
		buf.Myprintf("INDICES OF %v", node.IndicesOfOpt)
		if node.BetweenBoundOpt != nil {
			buf.Myprintf(" %v", node.BetweenBoundOpt)
		}
	} else if node.ValuesOf != nil {
		buf.Myprintf("VALUES OF %v", node.ValuesOf)
	}
}

func (node *BoundsClause) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.IndicesOfOpt, node.BetweenBoundOpt, node.ValuesOf); err != nil {
		return err
	}
	return nil
}

func (*ForAllStatement) PLStatement() {}

type ForAllStatement struct {
	IndexName         *IndexName
	BoundsClause      *BoundsClause
	SqlStatement      *SqlStatement
	SaveExceptionsOpt string
}

func (node *ForAllStatement) Format(buf *TrackedBuffer) {
	buf.Myprintf("FORALL %v IN %v %v", node.IndexName, node.BoundsClause, node.SqlStatement)
	if node.SaveExceptionsOpt != "" {
		buf.Myprintf(" %s", node.SaveExceptionsOpt)
	}
}

func (node *ForAllStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.IndexName, node.BoundsClause, node.SqlStatement); err != nil {
		return err
	}
	return nil
}

func (*NullStatement) PLStatement() {}

type NullStatement struct {
	Null string
}

func (node *NullStatement) Format(buf *TrackedBuffer) {
	buf.Myprintf("NULL;")
}

func (node *NullStatement) WalkSubtree(visit Visit) error {
	return nil
}

func (*RaiseStatement) PLStatement() {}

type RaiseStatement struct {
	ExcetionNameOpt string
}

func (node *RaiseStatement) Format(buf *TrackedBuffer) {
	buf.Myprintf("RAISE")
	if node.ExcetionNameOpt != "" {
		buf.Myprintf(" %s", node.ExcetionNameOpt)
	}
	buf.Myprintf(";")
}

func (node *RaiseStatement) WalkSubtree(visit Visit) error {
	return nil
}

func (*ReturnStatement) PLStatement() {}

type ReturnStatement struct {
	ExpressionOpt *PlsqlExpression
}

func (node *ReturnStatement) Format(buf *TrackedBuffer) {
	buf.Myprintf("RETURN")
	if node.ExpressionOpt != nil {
		buf.Myprintf(" %v", node.ExpressionOpt)
	}
	buf.Myprintf(";")
}

func (node *ReturnStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.ExpressionOpt); err != nil {
		return err
	}
	return nil
}

type SimpleCaseWhenPart struct {
	WhenCondition     *PlsqlExpression
	SeqOfStatementOpt *SeqOfStatement
	ExpressionOpt     *PlsqlExpression
}

func (node *SimpleCaseWhenPart) Format(buf *TrackedBuffer) {
	buf.Myprintf("WHEN %v THEN", node.WhenCondition)
	if node.SeqOfStatementOpt != nil {
		buf.Myprintf(" %v", node.SeqOfStatementOpt)
	} else if node.ExpressionOpt != nil {
		buf.Myprintf(" %v;", node.ExpressionOpt)
	}
}

func (node *SimpleCaseWhenPart) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.WhenCondition, node.SeqOfStatementOpt, node.ExpressionOpt); err != nil {
		return err
	}
	return nil
}

type SimpleCaseWhenPartList []*SimpleCaseWhenPart

func (node SimpleCaseWhenPartList) Format(buf *TrackedBuffer) {
	for i, when := range node {
		if i == 0 {
			buf.Myprintf("%v", when)
		} else {
			buf.Myprintf(" %v", when)
		}
	}
}

func (node SimpleCaseWhenPartList) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, when := range node {
		if err := Walk(visit, when); err != nil {
			return err
		}
	}
	return nil
}

type SearchCaseWhenPart struct {
	WhenCondition     *PlsqlExpression
	SeqOfStatementOpt *SeqOfStatement
	ExpressionOpt     *PlsqlExpression
}

func (node *SearchCaseWhenPart) Format(buf *TrackedBuffer) {
	buf.Myprintf("WHEN %v THEN", node.WhenCondition)
	if node.SeqOfStatementOpt != nil {
		buf.Myprintf(" %v", node.SeqOfStatementOpt)
	} else if node.ExpressionOpt != nil {
		buf.Myprintf(" %v;", node.ExpressionOpt)
	}
}

func (node *SearchCaseWhenPart) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.WhenCondition, node.SeqOfStatementOpt, node.ExpressionOpt); err != nil {
		return err
	}
	return nil
}

type SearchCaseWhenPartList []*SearchCaseWhenPart

func (node SearchCaseWhenPartList) Format(buf *TrackedBuffer) {
	for i, when := range node {
		if i == 0 {
			buf.Myprintf("%v", when)
		} else {
			buf.Myprintf(" %v", when)
		}
	}
}

func (node SearchCaseWhenPartList) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, when := range node {
		if err := Walk(visit, when); err != nil {
			return err
		}
	}
	return nil
}

type CaseElsePart struct {
	SeqOfStatementOpt *SeqOfStatement
	ExpressionOpt     *PlsqlExpression
}

func (node *CaseElsePart) Format(buf *TrackedBuffer) {
	buf.Myprintf("ELSE")
	if node.SeqOfStatementOpt != nil {
		buf.Myprintf(" %v", node.SeqOfStatementOpt)
	} else if node.ExpressionOpt != nil {
		buf.Myprintf(" %v;", node.ExpressionOpt)
	}
}

func (node *CaseElsePart) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.SeqOfStatementOpt, node.ExpressionOpt); err != nil {
		return err
	}
	return nil
}

type SimpleCaseStatement struct {
	StartLabelNameOpt string
	CaseCondition     *PlsqlExpression
	CaseWhenPart      SimpleCaseWhenPartList
	CaseElsePartOpt   *CaseElsePart
	EndLabelName      string
}

func (node *SimpleCaseStatement) Format(buf *TrackedBuffer) {
	if node.StartLabelNameOpt != "" {
		buf.Myprintf("%s ", node.StartLabelNameOpt)
	}
	buf.Myprintf("CASE %v %v", node.CaseCondition, node.CaseWhenPart)
	if node.CaseElsePartOpt != nil {
		buf.Myprintf(" %v", node.CaseElsePartOpt)
	}
	buf.Myprintf(" END CASE")
	if node.EndLabelName != "" {
		buf.Myprintf(" %s", node.EndLabelName)
	}
}

func (node *SimpleCaseStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.CaseCondition, node.CaseWhenPart, node.CaseElsePartOpt); err != nil {
		return err
	}
	return nil
}

type SearchCaseStatement struct {
	StartLabelNameOpt string
	CaseWhenPart      SearchCaseWhenPartList
	CaseElsePartOpt   *CaseElsePart
	EndLabelName      string
}

func (node *SearchCaseStatement) Format(buf *TrackedBuffer) {
	if node.StartLabelNameOpt != "" {
		buf.Myprintf("%s ", node.StartLabelNameOpt)
	}
	buf.Myprintf("CASE %v", node.CaseWhenPart)
	if node.CaseElsePartOpt != nil {
		buf.Myprintf(" %v", node.CaseElsePartOpt)
	}
	buf.Myprintf(" END CASE")
	if node.EndLabelName != "" {
		buf.Myprintf(" %s", node.EndLabelName)
	}
}

func (node *SearchCaseStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.CaseWhenPart, node.CaseElsePartOpt); err != nil {
		return err
	}
	return nil
}

func (*CaseStatement) PLStatement() {}

type CaseStatement struct {
	SimpleCaseOpt *SimpleCaseStatement
	SearchCaseOpt *SearchCaseStatement
}

func (node *CaseStatement) Format(buf *TrackedBuffer) {
	if node.SimpleCaseOpt != nil {
		buf.Myprintf("%v;", node.SimpleCaseOpt)
	} else if node.SearchCaseOpt != nil {
		buf.Myprintf("%v;", node.SearchCaseOpt)
	}
}

func (node *CaseStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.SimpleCaseOpt, node.SearchCaseOpt); err != nil {
		return err
	}
	return nil
}

type ExecuteImmediateSuffix interface {
	IsExecuteImmediateSuffix()
	SQLNode
}

type VariableName struct {
	CharSetNameOpt IdExpressionList
	Name           IdExpressionList
	//todo bind_variable
}

func (node *VariableName) Format(buf *TrackedBuffer) {
	if node.CharSetNameOpt != nil {
		buf.Myprintf("%v ", node.CharSetNameOpt)
	}
	buf.Myprintf("%v", node.Name)
}

func (node *VariableName) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.CharSetNameOpt, node.Name); err != nil {
		return err
	}
	return nil
}

type VariableNameList []*VariableName

func (node VariableNameList) Format(buf *TrackedBuffer) {
	for i, variable := range node {
		if i == 0 {
			buf.Myprintf("%v", variable)
		} else {
			buf.Myprintf(", %v", variable)
		}
	}
}

func (node VariableNameList) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, variable := range node {
		if err := Walk(visit, variable); err != nil {
			return err
		}
	}
	return nil
}

type IntoClause struct {
	BulkOpt       string
	VariableNames VariableNameList
}

func (node *IntoClause) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	if node.BulkOpt != "" {
		buf.Myprintf(" BULK COLLECT")
	}
	buf.Myprintf(" INTO %v", node.VariableNames)
}

func (node *IntoClause) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.VariableNames); err != nil {
		return err
	}
	return nil
}

type UsingElement struct {
	ParameterOpt  string
	SelectElement string
	AliasOpt      string
}

func (node *UsingElement) Format(buf *TrackedBuffer) {
	if node.ParameterOpt != "" {
		buf.Myprintf("%s ", node.ParameterOpt)
	}
	buf.Myprintf("%s", node.SelectElement)
	if node.AliasOpt != "" {
		buf.Myprintf(" AS %s", node.AliasOpt)
	}
}

func (node *UsingElement) WalkSubtree(visit Visit) error {
	return nil
}

type UsingElementList []*UsingElement

func (node UsingElementList) Format(buf *TrackedBuffer) {
	for i, ele := range node {
		if i == 0 {
			buf.Myprintf("%v", ele)
		} else {
			buf.Myprintf(", %v", ele)
		}
	}
}

func (node UsingElementList) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, ele := range node {
		if err := Walk(visit, ele); err != nil {
			return err
		}
	}
	return nil
}

type UsingClause struct {
	AllOpt          string
	UsingElementOpt UsingElementList
}

func (node *UsingClause) Format(buf *TrackedBuffer) {
	if node.AllOpt != "" {
		buf.Myprintf("USING *")
	} else if node.UsingElementOpt != nil {
		buf.Myprintf("USING %v", node.UsingElementOpt)
	}
}

func (node *UsingClause) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.UsingElementOpt); err != nil {
		return err
	}
	return nil
}

func (*IntoUsing) IsExecuteImmediateSuffix() {}

type IntoUsing struct {
	IntoClause *IntoClause
	UsingOpt   *UsingClause
}

func (node *IntoUsing) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v", node.IntoClause)
	if node.UsingOpt != nil {
		buf.Myprintf(" %v", node.UsingOpt)
	}
}

func (node *IntoUsing) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.IntoClause, node.UsingOpt); err != nil {
		return err
	}
	return nil
}

func (*DynamicReturn) IsExecuteImmediateSuffix() {}

type DynamicReturn struct {
	ReturnName string
	IntoClause *IntoClause
}

func (node *DynamicReturn) Format(buf *TrackedBuffer) {
	buf.Myprintf(" %s%v", node.ReturnName, node.IntoClause)
}

func (node *DynamicReturn) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.IntoClause); err != nil {
		return err
	}
	return nil
}

func (*UsingReturn) IsExecuteImmediateSuffix() {}

type UsingReturn struct {
	UsingClause *UsingClause
	ReturnOpt   *DynamicReturn
}

func (node *UsingReturn) Format(buf *TrackedBuffer) {
	buf.Myprintf(" %v", node.UsingClause)
	if node.ReturnOpt != nil {
		buf.Myprintf(" %v", node.ReturnOpt)
	}
}

func (node *UsingReturn) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.UsingClause, node.ReturnOpt); err != nil {
		return err
	}
	return nil
}

type ExecuteImmediate struct {
	StmtStr *PlsqlExpression
	Suffix  ExecuteImmediateSuffix
}

func (node *ExecuteImmediate) Format(buf *TrackedBuffer) {
	buf.Myprintf("EXECUTE IMMEDIATE %v", node.StmtStr)
	if node.Suffix != nil {
		buf.Myprintf("%v", node.Suffix)
	}
}

func (node *ExecuteImmediate) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.StmtStr, node.Suffix); err != nil {
		return err
	}
	return nil
}

type OpenStatement struct {
	CursorName          *VariableName
	PlsqlExpressionsOpt PlsqlExpressions
}

func (node *OpenStatement) Format(buf *TrackedBuffer) {
	buf.Myprintf("OPEN %s", node.CursorName)
	if node.PlsqlExpressionsOpt != nil {
		buf.Myprintf(" %v", node.PlsqlExpressionsOpt)
	}
}

func (node *OpenStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.CursorName, node.PlsqlExpressionsOpt); err != nil {
		return err
	}
	return nil
}

type FetchStatement struct {
	CursorName  *ColName
	IntoOpt     VariableNameList
	BulkIntoOpt VariableNameList
}

func (node *FetchStatement) Format(buf *TrackedBuffer) {
	buf.Myprintf("FETCH %v", node.CursorName)
	if node.IntoOpt != nil {
		buf.Myprintf(" INTO %v", node.IntoOpt)
	} else if node.BulkIntoOpt != nil {
		buf.Myprintf(" BULK COLLECT INTO %v", node.BulkIntoOpt)
	}
}

func (node *FetchStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.CursorName, node.IntoOpt, node.BulkIntoOpt); err != nil {
		return err
	}
	return nil
}

type OpenForStatement struct {
	VariableName    *VariableName
	SelStatementOpt *Select
	ExpressionOpt   *PlsqlExpression
	UsingClause     *UsingClause
}

func (node *OpenForStatement) Format(buf *TrackedBuffer) {
	buf.Myprintf("OPEN %v FOR", node.VariableName)
	if node.SelStatementOpt != nil {
		buf.Myprintf(" %v", node.SelStatementOpt)
	} else if node.ExpressionOpt != nil {
		buf.Myprintf(" %v", node.ExpressionOpt)
	}
	if node.UsingClause != nil {
		buf.Myprintf(" %v", node.UsingClause)
	}
}

func (node *OpenForStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.VariableName, node.SelStatementOpt, node.ExpressionOpt, node.UsingClause); err != nil {
		return err
	}
	return nil
}

type CursorManuStatement struct {
	CloseStatement   string
	OpenStatement    *OpenStatement
	FetchStatement   *FetchStatement
	OpenForStatement *OpenForStatement
}

func (node *CursorManuStatement) Format(buf *TrackedBuffer) {
	if node.CloseStatement != "" {
		buf.Myprintf("CLOSE %s", node.CloseStatement)
	} else if node.OpenStatement != nil {
		buf.Myprintf("%v", node.OpenStatement)
	} else if node.FetchStatement != nil {
		buf.Myprintf("%v", node.FetchStatement)
	} else {
		buf.Myprintf("%v", node.OpenForStatement)
	}
}

func (node *CursorManuStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.OpenStatement, node.FetchStatement, node.OpenForStatement); err != nil {
		return err
	}
	return nil
}

type SetTransCommand struct {
	ReadOpt      string
	IsolationOpt string
	UseOpt       ColIdent
	QuotedString string
}

func (node *SetTransCommand) Format(buf *TrackedBuffer) {
	buf.Myprintf("SET TRANSACTION")
	if node.ReadOpt != "" {
		buf.Myprintf(" READ %s", strings.ToUpper(node.ReadOpt))
	} else if node.IsolationOpt != "" {
		buf.Myprintf(" ISOLATION LEVEL %s", strings.ToUpper(node.IsolationOpt))
	} else if !node.UseOpt.IsEmpty() {
		buf.Myprintf(" USE ROLLBACK SEGMENT %v", node.UseOpt)
	}
	if node.QuotedString != "" {
		buf.Myprintf(" NAME %s", node.QuotedString)
	}
}

func (node *SetTransCommand) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.UseOpt); err != nil {
		return err
	}
	return nil
}

type ConstraintName struct {
	Prefix      ColIdent
	Expressions IdExpressionList
	LinkNameOpt string
}

func (node *ConstraintName) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v %v", node.Prefix, node.Expressions)
	if node.LinkNameOpt != "" {
		buf.Myprintf(" %s", node.LinkNameOpt)
	}
}

func (node *ConstraintName) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.Prefix, node.Expressions); err != nil {
		return err
	}
	return nil
}

type ConstraintNameList []*ConstraintName

func (node ConstraintNameList) Format(buf *TrackedBuffer) {
	var prefix string
	for _, constraint := range node {
		buf.Myprintf("%s%v", prefix, constraint)
		prefix = ", "
	}
}

func (node ConstraintNameList) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, constraint := range node {
		if err := Walk(visit, constraint); err != nil {
			return err
		}
	}
	return nil
}

type SetConstCommand struct {
	ConstraintOpt      string
	AllOpt             string
	ConstraintNamesOpt ConstraintNameList
	SuffixOpt          string
}

func (node *SetConstCommand) Format(buf *TrackedBuffer) {
	buf.Myprintf("SET %s", strings.ToUpper(node.ConstraintOpt))
	if node.AllOpt != "" {
		buf.Myprintf(" %s", strings.ToUpper(node.AllOpt))
	} else if node.ConstraintNamesOpt != nil {
		buf.Myprintf(" %v", node.ConstraintNamesOpt)
	}
	buf.Myprintf(" %s", strings.ToUpper(node.SuffixOpt))
}

func (node *SetConstCommand) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.ConstraintNamesOpt); err != nil {
		return err
	}
	return nil
}

type CommitStatement struct {
	WorkOpt       string
	CommentOpt    *PlsqlExpression
	ForceExprOpt  *PlsqlExpression
	ForceExprsOpt PlsqlExpressions
	ForceKeyword  string
	WriteClause   string
}

func (node *CommitStatement) Format(buf *TrackedBuffer) {
	buf.Myprintf("COMMIT")
	if node.WorkOpt != "" {
		buf.Myprintf(" WORK")
	}
	if node.CommentOpt != nil {
		buf.Myprintf(" COMMENT %v", node.CommentOpt)
	} else {
		buf.Myprintf(" FORCE")
		if node.ForceExprOpt != nil {
			buf.Myprintf(" CORRUPT_XID %v", node.ForceExprOpt)
		} else if node.ForceExprsOpt != nil {
			buf.Myprintf(" %v", node.ForceExprsOpt)
		} else {
			buf.Myprintf(" CORRUPT_XID_ALL")
		}
	}
	if node.WriteClause != "" {
		buf.Myprintf(" %s", strings.ToUpper(node.WriteClause))
	}

}

func (node *CommitStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.CommentOpt, node.ForceExprOpt, node.ForceExprsOpt); err != nil {
		return err
	}
	return nil
}

type Identifier struct {
	CharsetNameOpt IdExpressionList
	IdExpression   *IdExpression
}

func (node *Identifier) Format(buf *TrackedBuffer) {
	if node.CharsetNameOpt != nil {
		buf.Myprintf("INTRODUCER %v ", node.CharsetNameOpt)
	}
	buf.Myprintf("%v", node.IdExpression)
}

func (node *Identifier) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.CharsetNameOpt, node.IdExpression); err != nil {
		return err
	}
	return nil
}

type RollbackStatement struct {
	SavePointName *Identifier
	QuotedString  string
	WorkOpt       string
	SavePointOpt  string
}

func (node *RollbackStatement) Format(buf *TrackedBuffer) {
	buf.Myprintf("ROLLBACK")
	if node.WorkOpt != "" {
		buf.Myprintf(" WORK")
	}
	if node.SavePointName != nil {
		buf.Myprintf(" TO")
		if node.SavePointOpt != "" {
			buf.Myprintf(" SAVEPOINT")
		}
		buf.Myprintf(" %v", node.SavePointName)
	} else if node.QuotedString != "" {
		buf.Myprintf(" FORCE %s", node.QuotedString)
	}
}

func (node *RollbackStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.SavePointName); err != nil {
		return err
	}
	return nil
}

type SavePointStatement struct {
	SavePointName *Identifier
}

func (node *SavePointStatement) Format(buf *TrackedBuffer) {
	buf.Myprintf("SAVEPOINT %v", node.SavePointName)
}

func (node *SavePointStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.SavePointName); err != nil {
		return err
	}
	return nil
}

type TransCtrlStatement struct {
	SetTransCommand    *SetTransCommand
	SetConstCommand    *SetConstCommand
	CommitStatement    *CommitStatement
	RollbackStatement  *RollbackStatement
	SavePointStatement *SavePointStatement
}

func (node *TransCtrlStatement) Format(buf *TrackedBuffer) {
	if node.SetTransCommand != nil {
		buf.Myprintf("%v", node.SetTransCommand)
	} else if node.SetConstCommand != nil {
		buf.Myprintf("%v", node.SetConstCommand)
	} else if node.CommitStatement != nil {
		buf.Myprintf("%v", node.CommitStatement)
	} else if node.RollbackStatement != nil {
		buf.Myprintf("%v", node.RollbackStatement)
	} else {
		buf.Myprintf("%v", node.SavePointStatement)
	}
}

func (node *TransCtrlStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.SetTransCommand, node.SetConstCommand, node.CommitStatement, node.RollbackStatement, node.SavePointStatement); err != nil {
		return err
	}
	return nil
}

type DmlStatement struct {
	SelectStatement  SelectStatement
	UpdateStatement  Statement
	DeleteStatement  Statement
	InsertStatement  Statement
	ExplainStatement Statement
	// todo MergeStatement and LockTableStatement
}

func (node *DmlStatement) Format(buf *TrackedBuffer) {
	if node.SelectStatement != nil {
		buf.Myprintf("%v", node.SelectStatement)
	} else if node.UpdateStatement != nil {
		buf.Myprintf("%v", node.UpdateStatement)
	} else if node.DeleteStatement != nil {
		buf.Myprintf("%v", node.DeleteStatement)
	} else if node.InsertStatement != nil {
		buf.Myprintf("%v", node.InsertStatement)
	} else {
		buf.Myprintf("%v", node.ExplainStatement)
	}
}

func (node *DmlStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.SelectStatement, node.UpdateStatement, node.DeleteStatement, node.InsertStatement, node.ExplainStatement); err != nil {
		return err
	}
	return nil
}

func (*SqlStatement) PLStatement() {}

type SqlStatement struct {
	ExecuteImmediate    *ExecuteImmediate
	DmlStatement        *DmlStatement
	CursorManuStatement *CursorManuStatement
	TransCtrlStatement  *TransCtrlStatement
}

func (node *SqlStatement) Format(buf *TrackedBuffer) {
	if node.ExecuteImmediate != nil {
		buf.Myprintf("%v;", node.ExecuteImmediate)
	} else if node.DmlStatement != nil {
		buf.Myprintf("%v;", node.DmlStatement)
	} else if node.CursorManuStatement != nil {
		buf.Myprintf("%v;", node.CursorManuStatement)
	} else {
		buf.Myprintf("%v;", node.TransCtrlStatement)
	}
}

func (node *SqlStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.ExecuteImmediate, node.DmlStatement, node.CursorManuStatement, node.TransCtrlStatement); err != nil {
		return err
	}
	return nil
}

func (*FunctionCallStatement) PLStatement() {}

type FunctionCallStatement struct {
	RoutineName  ProcOrFuncName
	ArgumentsOpt SelectExprs
}

func (node *FunctionCallStatement) Format(buf *TrackedBuffer) {
	buf.Myprintf("CALL %v", node.RoutineName)
	if node.ArgumentsOpt != nil {
		buf.Myprintf("(%v)", node.ArgumentsOpt)
	}
	buf.Myprintf(";")
}

func (node *FunctionCallStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.RoutineName, node.ArgumentsOpt); err != nil {
		return err
	}
	return nil
}

func (*PipeRowStatement) PLStatement() {}

type PipeRowStatement struct {
	Expression *PlsqlExpression
}

func (node *PipeRowStatement) Format(buf *TrackedBuffer) {
	buf.Myprintf("PIPE ROW(%v);", node.Expression)
}

func (node *PipeRowStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.Expression); err != nil {
		return err
	}
	return nil
}

func (*LabelDeclaration) PLStatement() {}

type LabelDeclaration struct {
	LabelName string
}

func (node *LabelDeclaration) Format(buf *TrackedBuffer) {
	buf.Myprintf("<< %s >>", node.LabelName)
}

func (node *LabelDeclaration) WalkSubtree(visit Visit) error {
	return nil
}

type SeqOfStatement struct {
	PlStatements []PLStatement
}

func (node *SeqOfStatement) Format(buf *TrackedBuffer) {
	for i, plStatement := range node.PlStatements {
		if i != 0 {
			buf.Myprintf(" ")
		}
		buf.Myprintf("%v", plStatement)
	}
}

func (node *SeqOfStatement) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, plStatement := range node.PlStatements {
		if err := Walk(visit, plStatement); err != nil {
			return err
		}
	}
	return nil
}

func (node *SeqOfStatement) AddPlStatement(plsqlState PLStatement) {
	node.PlStatements = append(node.PlStatements, plsqlState)
}

type Body struct {
	SeqOfStatement       *SeqOfStatement
	ExceptionHandlersOpt ExceptionHandlerList
	EndNameOpt           string
}

func (node *Body) Format(buf *TrackedBuffer) {
	buf.Myprintf("BEGIN %v", node.SeqOfStatement)
	if node.ExceptionHandlersOpt != nil {
		buf.Myprintf("%v", node.ExceptionHandlersOpt)
	}
	buf.Myprintf(" END")
	if node.EndNameOpt != "" {
		buf.Myprintf(" %s", node.EndNameOpt)
	}
}

func (node *Body) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.SeqOfStatement, node.ExceptionHandlersOpt); err != nil {
		return err
	}
	return nil
}

type DeclareSpec struct {
	PLDeclaration []PLDeclaration
}

func (node *DeclareSpec) Format(buf *TrackedBuffer) {
	for _, dec := range node.PLDeclaration {
		buf.Myprintf(" %v;", dec)
	}
}

func (node *DeclareSpec) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, dec := range node.PLDeclaration {
		if err := Walk(visit, dec); err != nil {
			return err
		}
	}
	return nil
}

type Block struct {
	DeclareSpecsOpt *DeclareSpec
	Body            *Body
}

func (node *Block) Format(buf *TrackedBuffer) {
	if node.DeclareSpecsOpt != nil {
		buf.Myprintf(" DECLARE")
		buf.Myprintf("%v", node.DeclareSpecsOpt)
	}
	buf.Myprintf(" %v", node.Body)
}

func (node *Block) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.DeclareSpecsOpt, node.Body); err != nil {
		return err
	}
	return nil
}

type CSpec struct {
	// todo
}

func (node *CSpec) Format(buf *TrackedBuffer) {

}

func (node *CSpec) WalkSubtree(visit Visit) error {
	return nil
}

type CallSpec struct {
	JavaSpec string
	CSpec    *CSpec
}

// Format formats the node.
func (node *CallSpec) Format(buf *TrackedBuffer) {
	buf.Myprintf("LANGUAGE")
	if node.JavaSpec != "" {
		buf.Myprintf(" %s", node.JavaSpec)
	} else if node.CSpec != nil {
		buf.Myprintf(" %v", node.CSpec)
	}
}

func (node *CallSpec) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.CSpec); err != nil {
		return err
	}
	return nil
}

func (*ProcedureDefinition) PLPackageBody() {}

type ProcedureDefinition struct {
	ProcName      TableName
	ParametersOpt ParameterList
	Block         *Block
	CallSpecOpt   *CallSpec
	ExternalOpt   string
}

// Format formats the node.
func (node *ProcedureDefinition) Format(buf *TrackedBuffer) {
	buf.Myprintf("PROCEDURE %v", node.ProcName)
	if node.ParametersOpt != nil {
		buf.Myprintf(" (%v)", node.ParametersOpt)
	}
	buf.Myprintf(" IS")
	if node.Block != nil {
		buf.Myprintf("%v", node.Block)
	} else if node.CallSpecOpt != nil {
		buf.Myprintf(" %v", node.CallSpecOpt)
	} else if node.ExternalOpt != "" {
		buf.Myprintf(" %s", node.ExternalOpt)
	}
}

func (node *ProcedureDefinition) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.ProcName, node.ParametersOpt, node.Block, node.CallSpecOpt); err != nil {
		return err
	}
	return nil
}

type FuncReturnSuffix interface {
	IsFuncReturnSuffix()
	SQLNode
}

func (*InvokerRightsClause) IsFuncReturnSuffix() {}

type InvokerRightsClause struct {
	KeyWord string
}

// Format formats the node.
func (node *InvokerRightsClause) Format(buf *TrackedBuffer) {
	buf.Myprintf("%s", strings.ToUpper(node.KeyWord))
}

func (node *InvokerRightsClause) WalkSubtree(visit Visit) error {
	return nil
}

type ParenColumnList struct {
	ColumnList Columns
}

func (node *ParenColumnList) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v", node.ColumnList)
}

func (node *ParenColumnList) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.ColumnList); err != nil {
		return err
	}
	return nil
}

type StreamingClause struct {
	Type            string
	Expression      *PlsqlExpression
	ParenColumnList *ParenColumnList
}

func (node *StreamingClause) Format(buf *TrackedBuffer) {
	buf.Myprintf("%s %v BY %v", strings.ToUpper(node.Type), node.Expression, node.ParenColumnList)
}

func (node *StreamingClause) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.Expression, (node)); err != nil {
		return err
	}
	return nil
}

type PartitionByClause struct {
	Expression      *PlsqlExpression
	AnyOpt          string
	Type            string
	ParenColumnList *ParenColumnList
	StreamingClause *StreamingClause
}

// Format formats the node.
func (node *PartitionByClause) Format(buf *TrackedBuffer) {
	buf.Myprintf("(PARTITION %v BY", node.Expression)
	if node.AnyOpt != "" {
		buf.Myprintf(" %s", strings.ToUpper(node.AnyOpt))
	} else if node.ParenColumnList != nil {
		buf.Myprintf(" %s %v", strings.ToUpper(node.Type), node.ParenColumnList)
	}
	buf.Myprintf(")")
	if node.StreamingClause != nil {
		buf.Myprintf(" %v", node.StreamingClause)
	}
}

func (node *PartitionByClause) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.Expression, node.ParenColumnList, node.StreamingClause); err != nil {
		return err
	}
	return nil
}

func (*ParallelEnableClause) IsFuncReturnSuffix() {}

type ParallelEnableClause struct {
	PartitionByClause *PartitionByClause
}

// Format formats the node.
func (node *ParallelEnableClause) Format(buf *TrackedBuffer) {
	buf.Myprintf("PARALLEL_ENABLE")
	if node.PartitionByClause != nil {
		buf.Myprintf(" %v", node.PartitionByClause)
	}
}

func (node *ParallelEnableClause) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.PartitionByClause); err != nil {
		return err
	}
	return nil
}

type PartitionExtension struct {
	PartitionMode  string
	ForOpt         string
	ExpressionsOpt PlsqlExpressions
}

// Format formats the node.
func (node *PartitionExtension) Format(buf *TrackedBuffer) {
	buf.Myprintf("%s", strings.ToUpper(node.PartitionMode))
	if node.ForOpt != "" {
		buf.Myprintf(" %s", strings.ToUpper(node.ForOpt))
	}
	buf.Myprintf("(")
	if node.ExpressionsOpt != nil {
		buf.Myprintf("%v", node.ExpressionsOpt)
	}
	buf.Myprintf(")")
}

func (node *PartitionExtension) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.ExpressionsOpt); err != nil {
		return err
	}
	return nil
}

type TableViewName struct {
	Identifier         *Identifier
	IdExpressionOpt    *IdExpression
	LinkNameOpt        *Identifier
	PartitionExptenOpt *PartitionExtension
}

// Format formats the node.
func (node *TableViewName) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v", node.Identifier)
	if node.IdExpressionOpt != nil {
		buf.Myprintf(" %v", node.IdExpressionOpt)
	}
	if node.LinkNameOpt != nil {
		buf.Myprintf(" @%v", node.LinkNameOpt)
	} else if node.PartitionExptenOpt != nil {
		buf.Myprintf(" %v", node.PartitionExptenOpt)
	}
}

func (node *TableViewName) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.Identifier, node.IdExpressionOpt, node.LinkNameOpt, node.PartitionExptenOpt); err != nil {
		return err
	}
	return nil
}

type TableViewNameList []*TableViewName

// Format formats the node.
func (node TableViewNameList) Format(buf *TrackedBuffer) {
	var prefix string
	for _, tableview := range node {
		buf.Myprintf("%s%v", prefix, tableview)
		prefix = ", "
	}
}

func (node TableViewNameList) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, tableview := range node {
		if err := Walk(visit, tableview); err != nil {
			return err
		}
	}
	return nil
}

type ReliesOnPart struct {
	TableViewNames TableViewNameList
}

// Format formats the node.
func (node *ReliesOnPart) Format(buf *TrackedBuffer) {
	buf.Myprintf("RELIES_ON (%v)", node.TableViewNames)
}

func (node *ReliesOnPart) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.TableViewNames); err != nil {
		return err
	}
	return nil
}

func (*ResultCacheClause) IsFuncReturnSuffix() {}

type ResultCacheClause struct {
	ReliesOnPart *ReliesOnPart
}

// Format formats the node.
func (node *ResultCacheClause) Format(buf *TrackedBuffer) {
	buf.Myprintf("RESULT_CACHE")
	if node.ReliesOnPart != nil {
		buf.Myprintf(" %v", node.ReliesOnPart)
	}
}

func (node *ResultCacheClause) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.ReliesOnPart); err != nil {
		return err
	}
	return nil
}

func (*Deterministic) IsFuncReturnSuffix() {}

type Deterministic struct {
	KeyWord string
}

// Format formats the node.
func (node *Deterministic) Format(buf *TrackedBuffer) {
	buf.Myprintf("%s", strings.ToUpper(node.KeyWord))
}

func (node *Deterministic) WalkSubtree(visit Visit) error {
	return nil
}

type FuncReturnSuffixList []FuncReturnSuffix

// Format formats the node.
func (node FuncReturnSuffixList) Format(buf *TrackedBuffer) {
	var prefix string
	for _, suffix := range node {
		buf.Myprintf("%s%v", prefix, suffix)
		prefix = " "
	}
}

func (node FuncReturnSuffixList) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, suffix := range node {
		if err := Walk(visit, suffix); err != nil {
			return err
		}
	}
	return nil
}

func (*FunctionDefinition) PLPackageBody() {}

type FunctionDefinition struct {
	FuncName              TableName
	ParametersOpt         ParameterList
	ReturnType            *TypeSpec
	FuncReturnSuffixs     FuncReturnSuffixList
	PipelinedOpt          string
	Block                 *Block
	CallSpecOpt           *CallSpec
	PipelinedAggregateOpt string
	UsingImpleType        *IndexName
}

// Format formats the node.
func (node *FunctionDefinition) Format(buf *TrackedBuffer) {
	buf.Myprintf("FUNCTION %v", node.FuncName)
	if node.ParametersOpt != nil {
		buf.Myprintf(" (%v)", node.ParametersOpt)
	}
	buf.Myprintf(" RETURN %v", node.ReturnType)
	if node.FuncReturnSuffixs != nil {
		buf.Myprintf(" %v", node.FuncReturnSuffixs)
	}
	if node.PipelinedAggregateOpt != "" {
		buf.Myprintf(" %s", strings.ToUpper(node.PipelinedAggregateOpt))
		buf.Myprintf(" USING %v", node.UsingImpleType)
	} else {
		if node.PipelinedOpt != "" {
			buf.Myprintf(" %s", strings.ToUpper(node.PipelinedOpt))
		}
		buf.Myprintf(" IS")
		if node.Block != nil {
			buf.Myprintf("%v", node.Block)
		} else if node.CallSpecOpt != nil {
			buf.Myprintf(" %v", node.CallSpecOpt)
		}
	}
}

func (node *FunctionDefinition) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.FuncName, node.ParametersOpt, node.ReturnType, node.FuncReturnSuffixs, node.UsingImpleType, node.Block, node.CallSpecOpt); err != nil {
		return err
	}
	return nil
}

// Use in PackageSpec
type PLDeclaration interface {
	PLDeclaration()
	SQLNode
}

// create package stmt in plsql
type PackageSpec struct {
	PackageName      TableName
	IsPackageBody    bool
	InvokerRightsOpt string
	PLDeclaration    []PLDeclaration
}

func (node *PackageSpec) Format(buf *TrackedBuffer) {
	if node.InvokerRightsOpt != "" {
		buf.Myprintf("%v %s IS ", node.PackageName, node.InvokerRightsOpt)
	} else {
		buf.Myprintf("%v IS ", node.PackageName)
	}
	for _, dec := range node.PLDeclaration {
		buf.Myprintf("%v; ", dec)
	}
	buf.Myprintf("END %v;", node.PackageName)
}

func (node *PackageSpec) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.PackageName); err != nil {
		return err
	}
	for _, dec := range node.PLDeclaration {
		if err := Walk(visit, dec); err != nil {
			return err
		}
	}
	return nil
}

// Use in PackageBody
// Including []PLDeclaration and ProcedureDefinition and FunctionDefinition
type PLPackageBody interface {
	PLPackageBody()
	SQLNode
}

// create package body stmt in plsql
type PackageBody struct {
	PackageName   TableName
	PLPackageBody []PLPackageBody
	SuffixBodyOpt *Body
}

func (node *PackageBody) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v IS ", node.PackageName)
	for _, body := range node.PLPackageBody {
		buf.Myprintf("%v; ", body)
	}
	if node.SuffixBodyOpt != nil {
		buf.Myprintf("%v;", node.SuffixBodyOpt)
	} else {
		buf.Myprintf("END %v;", node.PackageName)
	}
}

func (node *PackageBody) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.PackageName, node.SuffixBodyOpt); err != nil {
		return err
	}
	for _, body := range node.PLPackageBody {
		if err := Walk(visit, body); err != nil {
			return err
		}
	}
	return nil
}

type PackageAlterBody struct {
	PackageName           TableName
	OptDebug              string
	OptSpecification      string
	OptCompilerParameters CompilerParameterList
	OptReuseSettings      string
}

func (node *PackageAlterBody) Format(buf *TrackedBuffer) {
	buf.Myprintf("PACKAGE %v COMPILE", node.PackageName)
	if node.OptDebug != "" {
		buf.Myprintf(" DEBUG")
	}
	if node.OptSpecification != "" {
		buf.Myprintf(" %s", strings.ToUpper(node.OptSpecification))
	}
	if node.OptCompilerParameters != nil {
		buf.Myprintf(" %v", node.OptCompilerParameters)
	}
	if node.OptReuseSettings != "" {
		buf.Myprintf(" %s", strings.ToUpper(node.OptReuseSettings))
	}
	buf.Myprintf(";")
}

func (node *PackageAlterBody) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.PackageName, node.OptCompilerParameters); err != nil {
		return err
	}
	return nil
}

// create function stmt in plsql
type FunctionBody struct {
	FuncName              TableName
	ParametersOpt         ParameterList
	ReturnType            *TypeSpec
	FuncReturnSuffixs     FuncReturnSuffixList
	PipelinedOpt          string
	Block                 *Block
	CallSpecOpt           *CallSpec
	PipelinedAggregateOpt string
	UsingImpleType        *IndexName
}

func (node *FunctionBody) Format(buf *TrackedBuffer) {
	buf.Myprintf("FUNCTION %v", node.FuncName)
	if node.ParametersOpt != nil {
		buf.Myprintf(" (%v)", node.ParametersOpt)
	}
	buf.Myprintf(" RETURN %v", node.ReturnType)
	if node.FuncReturnSuffixs != nil {
		buf.Myprintf(" %v", node.FuncReturnSuffixs)
	}
	if node.PipelinedAggregateOpt != "" {
		buf.Myprintf(" %s", strings.ToUpper(node.PipelinedAggregateOpt))
		buf.Myprintf(" USING %v;", node.UsingImpleType)
	} else {
		if node.PipelinedOpt != "" {
			buf.Myprintf(" %s", strings.ToUpper(node.PipelinedOpt))
		}
		buf.Myprintf(" IS")
		if node.Block != nil {
			buf.Myprintf("%v;", node.Block)
		} else if node.CallSpecOpt != nil {
			buf.Myprintf(" %v;", node.CallSpecOpt)
		}
	}
}

func (node *FunctionBody) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.FuncName, node.ParametersOpt, node.ReturnType, node.FuncReturnSuffixs, node.UsingImpleType, node.Block, node.CallSpecOpt); err != nil {
		return err
	}
	return nil
}

type CompilerParameter struct {
	Identifier *Identifier
	Expression *PlsqlExpression
}

func (node *CompilerParameter) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v = %v", node.Identifier, node.Expression)
}

func (node *CompilerParameter) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.Identifier, node.Expression); err != nil {
		return err
	}
	return nil
}

type CompilerParameterList []*CompilerParameter

func (node CompilerParameterList) Format(buf *TrackedBuffer) {
	var prefix string
	for _, param := range node {
		buf.Myprintf("%s%v", prefix, param)
		prefix = " "
	}
}

func (node CompilerParameterList) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, param := range node {
		if err := Walk(visit, param); err != nil {
			return err
		}
	}
	return nil
}

type FunctionAlterBody struct {
	FuncName              TableName
	OptDebug              string
	OptCompilerParameters CompilerParameterList
	OptReuseSettings      string
}

func (node *FunctionAlterBody) Format(buf *TrackedBuffer) {
	buf.Myprintf("FUNCTION %v COMPILE", node.FuncName)
	if node.OptDebug != "" {
		buf.Myprintf(" DEBUG")
	}
	if node.OptCompilerParameters != nil {
		buf.Myprintf(" %v", node.OptCompilerParameters)
	}
	if node.OptReuseSettings != "" {
		buf.Myprintf(" %s", strings.ToUpper(node.OptReuseSettings))
	}
	buf.Myprintf(";")
}

func (node *FunctionAlterBody) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.FuncName, node.OptCompilerParameters); err != nil {
		return err
	}
	return nil
}

// create procedure stmt in plsql
type ProcedureBody struct {
	ProcName         TableName
	ParametersOpt    ParameterList
	InvokerRightsOpt string
	Block            *Block
	CallSpecOpt      *CallSpec
	ExternalOpt      string
}

func (node *ProcedureBody) Format(buf *TrackedBuffer) {
	buf.Myprintf("PROCEDURE %v", node.ProcName)
	if node.ParametersOpt != nil {
		buf.Myprintf(" (%v)", node.ParametersOpt)
	}
	if node.InvokerRightsOpt != "" {
		buf.Myprintf(" %s", strings.ToUpper(node.InvokerRightsOpt))
	}
	buf.Myprintf(" IS")
	if node.Block != nil {
		buf.Myprintf("%v;", node.Block)
	} else if node.CallSpecOpt != nil {
		buf.Myprintf(" %v;", node.CallSpecOpt)
	} else if node.ExternalOpt != "" {
		buf.Myprintf(" %s;", node.ExternalOpt)
	}
}

func (node *ProcedureBody) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.ProcName, node.ParametersOpt, node.Block, node.CallSpecOpt); err != nil {
		return err
	}
	return nil
}

type ProcedureAlterBody struct {
	procName              TableName
	OptDebug              string
	OptCompilerParameters CompilerParameterList
	OptReuseSettings      string
}

func (node *ProcedureAlterBody) Format(buf *TrackedBuffer) {
	buf.Myprintf("PROCEDURE %v COMPILE", node.procName)
	if node.OptDebug != "" {
		buf.Myprintf(" DEBUG")
	}
	if node.OptCompilerParameters != nil {
		buf.Myprintf(" %v", node.OptCompilerParameters)
	}
	if node.OptReuseSettings != "" {
		buf.Myprintf(" %s", strings.ToUpper(node.OptReuseSettings))
	}
	buf.Myprintf(";")
}

func (node *ProcedureAlterBody) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.procName, node.OptCompilerParameters); err != nil {
		return err
	}
	return nil
}

// SchemaSpec describe the procedure definition
type SchemaSpec struct {
	SchemaName TableIdent
	Options    string
}

func (node *SchemaSpec) Format(buf *TrackedBuffer) {
	if node.Options != "" {
		buf.Myprintf("%v %s", node.SchemaName, node.Options)
	} else {
		buf.Myprintf("%v", node.SchemaName)
	}
}

func (node *SchemaSpec) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.SchemaName,
	)
}

// ServerSpec describe the server definition
type ServerSpec struct {
	ServerName TableIdent
}

// SynonymSpec describe the sysnonym definition
type SynonymSpec struct {
	SynonymName TableName
	TargetName  TableName
}

// Partition strings
const (
	ReorganizeStr = "reorganize partition"
)

// PartitionSpec describe partition actions (for alter and create)
type PartitionSpec struct {
	Action      string
	Name        ColIdent
	Definitions []*PartitionDefinition
}

// Format formats the node.
func (node *PartitionSpec) Format(buf *TrackedBuffer) {
	switch node.Action {
	case ReorganizeStr:
		buf.Myprintf("%s %v into (", node.Action, node.Name)
		var prefix string
		for _, pd := range node.Definitions {
			buf.Myprintf("%s%v", prefix, pd)
			prefix = ", "
		}
		buf.Myprintf(")")
	default:
		panic("unimplemented")
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *PartitionSpec) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.Name); err != nil {
		return err
	}
	for _, def := range node.Definitions {
		if err := Walk(visit, def); err != nil {
			return err
		}
	}
	return nil
}

const (
	ChangeStr = "change"
)

type ChangeSpec struct {
	Action           string
	OldName          ColIdent
	ColumnDefinition *ColumnDefinition
	Order            string
}

// Format formats the node.
func (node *ChangeSpec) Format(buf *TrackedBuffer) {
	switch node.Action {
	case ChangeStr:
		buf.Myprintf("%s column %v %v %s", node.Action, node.OldName, node.ColumnDefinition, node.Order)
	default:
		panic("unimplemented")
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *ChangeSpec) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.ColumnDefinition,
	)
}

const (
	ModifyStr = "modify"
)

type ModifySpec struct {
	Action           string
	ColumnDefinition *ColumnDefinition
	Order            string
}

// Format formats the node.
func (node *ModifySpec) Format(buf *TrackedBuffer) {
	switch node.Action {
	case ModifyStr:
		buf.Myprintf("%s column %v %s", node.Action, node.ColumnDefinition, node.Order)
	default:
		panic("unimplemented")
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *ModifySpec) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.ColumnDefinition,
	)
}

// KunPartitionDefinition describes a very minimal partition definition
type KunPartitionDefinition struct {
	Name ColIdent
	Rows ValTuple
}

// Format formats the node
func (node *KunPartitionDefinition) Format(buf *TrackedBuffer) {
	buf.Myprintf("partition %v values less than (%v)", node.Name, node.Rows)
}

// WalkSubtree walks the nodes of the subtree.
func (node *KunPartitionDefinition) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Name,
		node.Rows,
	)
}

// CheckPartitionValid is to checkout partitions if valid
func CheckPartitionValid(ddl *DDL) error {
	if ddl.OptCreatePartition == nil {
		return nil
	}
	if ddl.OptCreatePartition.OptUsing.String() == RangePartition {
		return CheckRangePartitionValid(ddl.TableSpec.Columns, ddl.OptCreatePartition.Column, ddl.OptCreatePartition.Definitions)
	} else if ddl.OptCreatePartition.OptUsing.String() == IntervalPartition {
		return CheckIntervalPartitionValid(ddl.TableSpec.Columns, ddl.OptCreatePartition.Column, ddl.OptCreatePartition.Definitions)
	} else if ddl.OptCreatePartition.OptUsing.String() == ListPartition {
		return CheckListPartitionValid(ddl.TableSpec.Columns, ddl.OptCreatePartition.Column, ddl.OptCreatePartition.Definitions)
	} else if ddl.OptCreatePartition.OptUsing.String() == ReferencePartition {
		return CheckReferencePartitionValid(ddl.TableSpec.Constraints, ddl.OptCreatePartition.Column)
	} else if ddl.OptCreatePartition.OptUsing.String() == ReplicationPartition {
		return nil
	}
	return CheckOtherPartitionValid(ddl.TableSpec.Columns, ddl.OptCreatePartition.Column)
}

// CheckDuplicatedColumn is to check if there is have duplication column
func CheckDuplicatedColumn(Columns []ColIdent) error {
	var tmpColumn []ColIdent
	for _, column := range Columns {
		for _, t := range tmpColumn {
			if column.Equal(t) {
				return fmt.Errorf("duplicated column name %s", t.String())
			}
		}
		tmpColumn = append(tmpColumn, column)
	}
	return nil
}

// CheckDuplicatedName is to check if there is duplicated partition name
func CheckDuplicatedName(Definitions []*KunPartitionDefinition) error {
	var tmpName []ColIdent
	for _, def := range Definitions {
		for _, t := range tmpName {
			if t.Equal(def.Name) {
				return fmt.Errorf("duplicated partition name %s", t.String())
			}
		}
		tmpName = append(tmpName, def.Name)
	}
	return nil
}

// CheckValidColumn describes the column is valid or not
func CheckValidColumn(TableColumns []*ColumnDefinition, RangeColumns []ColIdent) error {
	if TableColumns == nil || RangeColumns == nil {
		return nil
	}
	for _, column := range RangeColumns {
		for k, t := range TableColumns {
			if column.Equal(t.Name) {
				break
			}
			if k == len(TableColumns)-1 {
				return fmt.Errorf("unknown column name %s", column.String())
			}
		}
	}
	return nil
}

// CheckValidConstraint describes the constraint if valid or not
func CheckValidConstraint(Constraints []*ConstraintDefinition, refColumn ColIdent) error {
	if Constraints == nil {
		return fmt.Errorf("no reference constraint")
	}
	for i, c := range Constraints {
		_, ok := c.Detail.(*PartitionKeyDetail)
		if !ok {
			if i == len(Constraints)-1 {
				return fmt.Errorf("reference constraint is not found")
			}
			continue
		}
		if !strings.EqualFold(c.Name, refColumn.String()) {
			return fmt.Errorf("unknown constraint name %s", refColumn.String())
		}
	}
	return nil
}

// CheckRangePartitionValid is to check the partition is valid or not for range partition
func CheckRangePartitionValid(TableColumns []*ColumnDefinition, RangeColumns []ColIdent, Definitions []*KunPartitionDefinition) error {
	if err := CheckValidColumn(TableColumns, RangeColumns); err != nil {
		return err
	}
	if err := CheckDuplicatedColumn(RangeColumns); err != nil {
		return err
	}
	if 1 >= len(Definitions) {
		return nil
	}
	if err := CheckDuplicatedName(Definitions); err != nil {
		return err
	}

	currValue := make([][]string, len(Definitions))
	var tmpValues []string
	colNum := len(Definitions[0].Rows)
	for j, partDef := range Definitions {
		if len(RangeColumns) != len(partDef.Rows) {
			return fmt.Errorf(" the number of values is incorrect")
		}
		tmpValues = []string{}
		isPartitionChecked := false
	LOOP:
		for i := 0; i < colNum; i++ {
			value := partDef.Rows[i]
			switch node := value.(type) {
			case *SQLVal:
				switch node.Type {
				case StrVal:
					tmpValues = append(tmpValues, string(node.Val))
					if i == colNum-1 {
						currValue[j] = tmpValues
					}
					if j != 0 && !isPartitionChecked {
						if strings.EqualFold(tmpValues[i], "maxvalue") {
							if strings.EqualFold(currValue[j-1][i], "maxvalue") {
								return fmt.Errorf("VALUES LESS THAN values must be strictly increasing for each partition")
							}
							if i == colNum-1 || j == len(Definitions)-1 {
								break LOOP
							}
							isPartitionChecked = true
						} else {
							if strings.EqualFold(currValue[j-1][i], "maxvalue") {
								return fmt.Errorf("VALUES LESS THAN values must be strictly increasing for each partition")
							}
							if strings.Compare(currValue[j-1][i], tmpValues[i]) < 0 {
								if i == colNum-1 || j == len(Definitions)-1 {
									break LOOP
								}
								isPartitionChecked = true
								// If strings.Compare(currValue[j-1][i], tmpValues[i])==0, should continue to compare the next column
								// but if it the last column, should return error
							} else if strings.Compare(currValue[j-1][i], tmpValues[i]) > 0 || (i == colNum-1) {
								return fmt.Errorf("VALUES LESS THAN values must be strictly increasing for each partition")
							}
						}
					}
				case IntVal:
					tmpValues = append(tmpValues, string(node.Val))
					if i == colNum-1 {
						currValue[j] = tmpValues
					}
					if j != 0 && !isPartitionChecked {
						var maxIntValue, newIntValue int64
						var err error
						if strings.EqualFold(currValue[j-1][i], "maxvalue") {
							maxIntValue = math.MaxInt64
						} else {
							maxIntValue, err = strconv.ParseInt(currValue[j-1][i], 0, 64)
							if err != nil {
								return err
							}
						}
						if strings.EqualFold(tmpValues[i], "maxvalue") {
							newIntValue = math.MaxInt64
						} else {
							newIntValue, err = strconv.ParseInt(tmpValues[i], 0, 64)
							if err != nil {
								return err
							}
						}
						if maxIntValue < newIntValue {
							if i == colNum-1 || j == len(Definitions)-1 {
								break LOOP
							}
							isPartitionChecked = true
						} else if maxIntValue > newIntValue || (i == colNum-1) {
							return fmt.Errorf("VALUES LESS THAN values must be strictly increasing for each partition")
						}
					}
				}
			default:
				return fmt.Errorf("invalid value type: %v", node)
			}
		}
	}
	return nil
}

// CheckIntervalPartitionValid is to check the partition is valid or not for interval partition
func CheckIntervalPartitionValid(TableColumns []*ColumnDefinition, RangeColumns []ColIdent, Definitions []*KunPartitionDefinition) error {
	if err := CheckValidColumn(TableColumns, RangeColumns); err != nil {
		return err
	}
	if len(RangeColumns) != 1 {
		return fmt.Errorf("interval partition support only one partitioning key column")
	}
	if err := CheckDuplicatedName(Definitions); err != nil {
		return err
	}

	first := true
	currentMaxValue := ""
	for _, partDef := range Definitions {
		value := partDef.Rows[0]
		if !IsNull(value) && !IsValue(value) {
			return fmt.Errorf("invalid value type: %v", value)
		}
		switch node := value.(type) {
		case *SQLVal:
			if strings.EqualFold(string(node.Val), "maxvalue") {
				return fmt.Errorf("interval partition can not support 'MAXVALUE'")
			}
			switch node.Type {
			case IntVal:
				if !first {
					maxIntValue, err := strconv.ParseInt(currentMaxValue, 0, 64)
					if err != nil {
						return err
					}
					newIntValue, err := strconv.ParseInt(string(node.Val), 0, 64)
					if err != nil {
						return err
					}
					if maxIntValue >= newIntValue {
						return fmt.Errorf("VALUES LESS THAN value must be strictly increasing for each partition")
					}
				}
				currentMaxValue = string(node.Val)
				first = false
			default:
				return fmt.Errorf("invalid value type: %v", node.Type)
			}
		default:
			return fmt.Errorf("invalid value type: %v", node)
		}
	}
	return nil
}

// CheckListPartitionValid is to check the partition is valid or not for list partition
func CheckListPartitionValid(TableColumns []*ColumnDefinition, ListColumns []ColIdent, Definitions []*KunPartitionDefinition) error {
	if len(ListColumns) != 1 {
		return fmt.Errorf("support only one partitioning key column")
	}
	if err := CheckValidColumn(TableColumns, ListColumns); err != nil {
		return err
	}
	if err := CheckDuplicatedName(Definitions); err != nil {
		return err
	}
	var tmpValues []string
	for _, partDef := range Definitions {
		for _, value := range partDef.Rows {
			switch node := value.(type) {
			case *SQLVal:
				for _, v := range tmpValues {
					if strings.EqualFold(v, string(node.Val)) {
						return fmt.Errorf("multiple definition of same constant in list partitioning: %s", v)
					}
				}
				tmpValues = append(tmpValues, string(node.Val))
			default:
				return fmt.Errorf("invalid value type: %v", node)
			}
		}
	}
	return nil
}

// CheckReferencePartitionValid is to check the partition is valid or not for reference partition
func CheckReferencePartitionValid(Constraints []*ConstraintDefinition, refColumns []ColIdent) error {
	if len(refColumns) != 1 {
		return fmt.Errorf("reference partition support only one partitioning key")
	}
	if err := CheckValidConstraint(Constraints, refColumns[0]); err != nil {
		return err
	}
	return nil
}

// CheckOtherPartitionValid is to check the partition is valid or not for other partition
func CheckOtherPartitionValid(TableColumns []*ColumnDefinition, partColumns []ColIdent) error {
	if len(partColumns) != 1 {
		return fmt.Errorf("support only one partitioning key column")
	}
	if err := CheckValidColumn(TableColumns, partColumns); err != nil {
		return err
	}
	return nil
}

// PartitionDefinition describes a very minimal partition definition
type PartitionDefinition struct {
	Name     ColIdent
	Limit    Expr
	Maxvalue bool
}

// Format formats the node
func (node *PartitionDefinition) Format(buf *TrackedBuffer) {
	if !node.Maxvalue {
		buf.Myprintf("partition %v values less than (%v)", node.Name, node.Limit)
	} else {
		buf.Myprintf("partition %v values less than (maxvalue)", node.Name)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *PartitionDefinition) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Name,
		node.Limit,
	)
}

// TableOption describes one option specified in the create table statement
type TableOption struct {
	OptionName            string
	OptionValue           string
	OptionStorage         string
	flagNotFormatCharset  bool
	flagNotFormatCollate  bool
	flagNotFormatFullText bool
}

func (node *TableOption) isCharsetOption() bool {
	return strings.Contains(strings.ToLower(node.OptionName), "character") || strings.Contains(strings.ToLower(node.OptionName), "charset")
}

func (node *TableOption) isCollateOption() bool {
	return strings.Contains(strings.ToLower(node.OptionName), "collate")
}

func (node *TableOption) isFullTextOption() bool {
	return strings.Contains(strings.ToLower(node.OptionName), "fulltext")
}

// Format formats the node
func (node *TableOption) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	if strings.EqualFold(node.OptionName, "connection") {
		buf.Myprintf("%s=%s", strings.ToUpper(node.OptionName), node.OptionValue)
	} else if strings.EqualFold(node.OptionName, "tablespace") {
		buf.Myprintf("%s %s%s", strings.ToUpper(node.OptionName), node.OptionValue, node.OptionStorage)
	} else if node.isCharsetOption() && node.flagNotFormatCharset {
		return
	} else if node.isCollateOption() && node.flagNotFormatCollate {
		return
	} else if node.isFullTextOption() && node.flagNotFormatFullText {
		return
	} else {
		buf.Myprintf("%s=%s", strings.ToUpper(node.OptionName), node.OptionValue)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *TableOption) WalkSubtree(visit Visit) error {
	return nil
}

// TableOptions is a list of TableOption, separated by space
type TableOptions []*TableOption

// Format formats the node
func (node TableOptions) Format(buf *TrackedBuffer) {
	if len(node) == 0 {
		return
	}
	for _, o := range node {
		if o == nil {
			continue
		}
		buf.Myprintf(" %v", o)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node TableOptions) WalkSubtree(visit Visit) error {
	for _, n := range node {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}
	return nil
}

// DontFormatCharsetCollate set the flag when formatting
func (node TableOptions) DontFormatCharsetCollate() {
	for _, o := range node {
		if o.isCharsetOption() {
			o.flagNotFormatCharset = true
		} else if o.isCollateOption() {
			o.flagNotFormatCollate = true
		}
	}
}

// DontFormatTextIndex set the flag when formatting
func (node TableOptions) DontFormatFullTextIndex() {
	for _, o := range node {
		if o.isFullTextOption() {
			o.flagNotFormatFullText = true
		}
	}
}

// Find returns the option value named by optionName
func (node TableOptions) Find(optionName string) string {
	for _, o := range node {
		if strings.EqualFold(o.OptionName, optionName) {
			return strings.ToLower(o.OptionValue)
		}
	}

	return ""
}

// Remove delete the option named by optionName if exist
func (node TableOptions) Remove(optionName string) {
	for i, o := range node {
		if strings.EqualFold(o.OptionName, optionName) {
			node[i] = nil
		}
	}
}

// TableSpec describes the structure of a table from a CREATE TABLE statement
type TableSpec struct {
	Columns     []*ColumnDefinition
	Indexes     []*IndexDefinition
	Constraints []*ConstraintDefinition
	Options     TableOptions
	// allChecks is true when all of constraints of this table is check constraint
	allChecks             bool
	flagNotFormatFullText bool
}

// GetColumnTypeByName return column type
func (ts *TableSpec) GetColumnTypeByName(name string) (string, error) {
	if ts == nil {
		return "", errors.New(errors.ErrInvalidArgument, "ts should be nil")
	}
	for _, column := range ts.Columns {
		if column != nil && column.Name.EqualString(name) {
			return column.Type.Type, nil
		}
	}
	return "", errors.New(errors.ErrTblColNotFound, fmt.Sprintf("col[%s] not found", name))
}

// Format formats the node.
func (ts *TableSpec) Format(buf *TrackedBuffer) {
	buf.Myprintf("(\n")
	for i, col := range ts.Columns {
		if i == 0 {
			buf.Myprintf("\t%v", col)
		} else {
			buf.Myprintf(",\n\t%v", col)
		}
	}
	for _, idx := range ts.Indexes {
		if idx.Info != nil && strings.HasPrefix(strings.ToLower(idx.Info.Type), "fulltext") && ts.flagNotFormatFullText {
			continue
		}
		buf.Myprintf(",\n\t%v", idx)
	}
	comma := ","
	if ts.Constraints != nil && ts.allChecks && !FormatCheckConstraint {
		comma = ""
	}
	for _, c := range ts.Constraints {
		buf.Myprintf("%s\n\t%v", comma, c)
	}
	buf.Myprintf("\n)%v", ts.Options)
}

// DontFormatFullText will set the table spce not to for all information about fulltext key
func (ts *TableSpec) DontFormatFullText() {
	ts.flagNotFormatFullText = true
	ts.Options.DontFormatFullTextIndex()
}

// AddColumn appends the given column to the list in the spec
func (ts *TableSpec) AddColumn(cd *ColumnDefinition) {
	ts.Columns = append(ts.Columns, cd)
}

// AddIndex appends the given index to the list in the spec
func (ts *TableSpec) AddIndex(id *IndexDefinition) {
	ts.Indexes = append(ts.Indexes, id)
}

// AddConstraint appends the given index to the list in the spec
func (ts *TableSpec) AddConstraint(cd *ConstraintDefinition) {
	if _, ok := cd.Detail.(*CheckCondition); !ok {
		ts.allChecks = false
	}
	ts.Constraints = append(ts.Constraints, cd)
}

// WalkSubtree walks the nodes of the subtree.
func (ts *TableSpec) WalkSubtree(visit Visit) error {
	if ts == nil {
		return nil
	}

	for _, n := range ts.Columns {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}

	for _, n := range ts.Indexes {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}

	for _, n := range ts.Constraints {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}

	return nil
}

// OptLike works for create table xxx like xxx
type OptLike struct {
	LikeTable TableName
}

// Format formats the node.
func (node *OptLike) Format(buf *TrackedBuffer) {
	buf.Myprintf("like %v", node.LikeTable)
}

// WalkSubtree walks the nodes of the subtree.
func (node *OptLike) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(visit, node.LikeTable)
}

// OptLocate works for locating the table
type OptLocate struct {
	Location string
}

// Format formats the node.
func (node *OptLocate) Format(buf *TrackedBuffer) {
}

// WalkSubtree walks the nodes of the subtree.
func (node *OptLocate) WalkSubtree(visit Visit) error {
	return nil
}

// OptCreatePartition works for create table xxx partition by xxx using xxx
type OptCreatePartition struct {
	Column      []ColIdent
	OptUsing    ColIdent
	ShardFunc   ColIdent
	OptInterval Expr
	Definitions []*KunPartitionDefinition
}

// Format formats the node.
func (node *OptCreatePartition) Format(buf *TrackedBuffer) {
}

// WalkSubtree walks the nodes of the subtree.
func (node *OptCreatePartition) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, p := range node.Column {
		err := Walk(visit, p)
		if err != nil {
			return err
		}
	}
	if err := Walk(visit, node.OptUsing); err != nil {
		return err
	}
	if err := Walk(visit, node.ShardFunc); err != nil {
		return err
	}
	for _, def := range node.Definitions {
		if err := Walk(visit, def); err != nil {
			return err
		}
	}
	return nil
}

// OptPartition records partitions in DML and TRUNCATE table/ALTER table DROP PARTITION/ALTER table RENAME PARTITION
type OptPartition struct {
	Action  string
	Name    TableIdent
	NewName TableIdent
}

// IndexSpec works for create global index
type IndexSpec struct {
	IdxDef *IndexDefinition
	Global bool
}

// Format formats the node.
func (node *IndexSpec) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v", node.IdxDef)
}

// WalkSubtree walks the nodes of the subtree.
func (node *IndexSpec) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	Walk(visit, node.IdxDef)
	return nil
}

type ColumnDetail struct {
	ColName  ColIdent
	IndexLen string
}

func NewColumnDetail(colident ColIdent, indexlen string) ColumnDetail {
	return ColumnDetail{
		ColName:  colident,
		IndexLen: indexlen,
	}
}

// Format formats the node.
func (node ColumnDetail) Format(buf *TrackedBuffer) {
	if node.IndexLen == "" {
		buf.Myprintf("%v", node.ColName)
	} else {
		buf.Myprintf("%v%s%s%s", node.ColName, "(", node.IndexLen, ")")
	}
}

func (node ColumnDetail) WalkSubtree(visit Visit) error {
	return nil
}

// ColumnDefinition describes a column in a CREATE TABLE statement
type ColumnDefinition struct {
	Name ColIdent
	Type ColumnType
}

// Format formats the node.
func (col *ColumnDefinition) Format(buf *TrackedBuffer) {
	buf.Myprintf("`%s` %v", col.Name.String(), &col.Type)
}

// WalkSubtree walks the nodes of the subtree.
func (col *ColumnDefinition) WalkSubtree(visit Visit) error {
	if col == nil {
		return nil
	}
	return Walk(
		visit,
		col.Name,
		&col.Type,
	)
}

type ColumnAttrList []ColumnAttr

func (node ColumnAttrList) parseDefault() (*SQLVal, error) {
	length := len(node)
	for i := length - 1; i >= 0; i-- {
		n := node[i]
		if n == nil {
			return nil, errors.New(errors.ErrUnexpected, "empty column attribute")
		}
		if def, ok := n.(*DefaultAttr); ok {
			return def.attr, nil
		}
	}
	return nil, nil
}

func (node ColumnAttrList) parseNull() (BoolVal, error) {
	length := len(node)
	for i := length - 1; i >= 0; i-- {
		n := node[i]
		if n == nil {
			return false, errors.New(errors.ErrUnexpected, "empty column attribute")
		}
		if def, ok := n.(*NullAttr); ok {
			return def.attr, nil
		}
	}
	return false, nil
}

func (node ColumnAttrList) parseOnUpdate() (*SQLVal, error) {
	length := len(node)
	for i := length - 1; i >= 0; i-- {
		n := node[i]
		if n == nil {
			return nil, errors.New(errors.ErrUnexpected, "empty column attribute")
		}
		if def, ok := n.(*OnUpdateAttr); ok {
			return def.attr, nil
		}
	}
	return nil, nil
}

func (node ColumnAttrList) parseAutoInc() (BoolVal, error) {
	length := len(node)
	for i := length - 1; i >= 0; i-- {
		n := node[i]
		if n == nil {
			return false, errors.New(errors.ErrUnexpected, "empty column attribute")
		}
		if def, ok := n.(*AutoIncAttr); ok {
			return def.attr, nil
		}
	}
	return false, nil
}

func (node ColumnAttrList) parseKeyOpt() (ColumnKeyOption, error) {
	length := len(node)
	for i := length - 1; i >= 0; i-- {
		n := node[i]
		if n == nil {
			return ColKeyNone, errors.New(errors.ErrUnexpected, "empty column attribute")
		}
		if def, ok := n.(ColumnKeyOption); ok {
			return def, nil
		}
	}
	return ColKeyNone, nil
}

func (node ColumnAttrList) parseComment() (*SQLVal, error) {
	length := len(node)
	for i := length - 1; i >= 0; i-- {
		n := node[i]
		if n == nil {
			return nil, errors.New(errors.ErrUnexpected, "empty column attribute")
		}
		if def, ok := n.(*ColumnComment); ok {
			return def.attr, nil
		}
	}
	return nil, nil
}

func (node ColumnAttrList) parseConstraint() (*ConstraintDefinition, error) {
	length := len(node)
	// support many constraint in column attribute list, but the last is in effective
	var cc *ColumnConstraint
	items := make([]*ConstraintStateItem, 0)
	for i := length - 1; i >= 0; i-- {
		n := node[i]
		if n == nil {
			return nil, errors.New(errors.ErrUnexpected, "empty column attribute")
		}
		if def, ok := n.(*ColumnConstraint); ok && cc == nil {
			cc = def
		} else if item, ok := n.(*ConstraintStateItem); ok {
			items = append(items, item)
		}
	}
	if cc == nil {
		return nil, nil
	}
	item := mergeConstraintState(items)
	return NewConstraintDef(cc.Name, cc.Detail, item), nil
}

type ColumnAttr interface {
	iColumnAttr()
}

func (node *DefaultAttr) iColumnAttr()         {}
func (node *NullAttr) iColumnAttr()            {}
func (node *OnUpdateAttr) iColumnAttr()        {}
func (node *AutoIncAttr) iColumnAttr()         {}
func (node *ColumnComment) iColumnAttr()       {}
func (node ColumnKeyOption) iColumnAttr()      {}
func (node *ColumnConstraint) iColumnAttr()    {}
func (node *ConstraintStateItem) iColumnAttr() {}

type DefaultAttr struct {
	attr *SQLVal
}

type NullAttr struct {
	attr BoolVal
}

func (node *NullAttr) GetAttr() BoolVal {
	return node.attr
}

type OnUpdateAttr struct {
	attr *SQLVal
}

type AutoIncAttr struct {
	attr BoolVal
}

type ColumnComment struct {
	attr *SQLVal
}

func (node *ColumnComment) GetAttr() *SQLVal {
	return node.attr
}

type ColumnConstraint struct {
	Name   string
	Detail ConstraintDetail
}

// ColumnType represents a sql type in a CREATE TABLE statement
// All optional fields are nil if not specified
type ColumnType struct {
	// The base type string
	Type string

	// Generated column options
	Generated BoolVal
	As        Expr
	Stored    BoolVal

	// Generic field options.
	NotNull       BoolVal
	Autoincrement BoolVal
	Default       *SQLVal
	OnUpdate      *SQLVal
	Comment       *SQLVal

	// Numeric field options
	Length   *SQLVal
	Unsigned BoolVal
	Zerofill BoolVal
	Scale    *SQLVal

	// Text field options
	Charset string
	Collate string

	// Enum values
	EnumValues []string

	//Spatial type
	Spatial BoolVal

	// Key specification
	KeyOpt ColumnKeyOption

	// Constraint
	Constraint *ConstraintDefinition
}

// Format returns a canonical string representation of the type and all relevant options
func (ct *ColumnType) Format(buf *TrackedBuffer) {
	buf.Myprintf("%s", ct.Type)

	if ct.Length != nil && ct.Scale != nil {
		buf.Myprintf("(%v,%v)", ct.Length, ct.Scale)

	} else if ct.Length != nil {
		buf.Myprintf("(%v)", ct.Length)
	}

	if ct.EnumValues != nil {
		buf.Myprintf("(%s)", strings.Join(ct.EnumValues, ", "))
	}

	opts := make([]string, 0, 16)
	if ct.Unsigned {
		opts = append(opts, keywordStrings[UNSIGNED])
	}
	if ct.Zerofill {
		opts = append(opts, keywordStrings[ZEROFILL])
	}
	if ct.Charset != "" {
		opts = append(opts, keywordStrings[CHARACTER], keywordStrings[SET], ct.Charset)
	}
	if ct.Collate != "" {
		opts = append(opts, keywordStrings[COLLATE], ct.Collate)
	}
	if ct.Generated {
		opts = append(opts, keywordStrings[AS], "(", String(ct.As), ")")
		if ct.Stored {
			opts = append(opts, keywordStrings[STORED])
		} else {
			opts = append(opts, keywordStrings[VIRTUAL])
		}
	}
	if ct.Default != nil {
		opts = append(opts, keywordStrings[DEFAULT], String(ct.Default))
	}
	if ct.NotNull {
		opts = append(opts, keywordStrings[NOT], keywordStrings[NULL])
	}
	if ct.OnUpdate != nil {
		opts = append(opts, keywordStrings[ON], keywordStrings[UPDATE], String(ct.OnUpdate))
	}
	if ct.Autoincrement {
		opts = append(opts, keywordStrings[AUTO_INCREMENT])
	}
	if ct.Comment != nil {
		opts = append(opts, keywordStrings[COMMENT_KEYWORD], String(ct.Comment))
	}
	if ct.KeyOpt == ColKeyPrimary {
		opts = append(opts, keywordStrings[PRIMARY], keywordStrings[KEY])
	}
	if ct.KeyOpt == ColKeyUnique {
		opts = append(opts, keywordStrings[UNIQUE])
	}
	if ct.KeyOpt == ColKeySpatialKey {
		opts = append(opts, keywordStrings[SPATIAL], keywordStrings[KEY])
	}
	if ct.KeyOpt == ColKeyUniqueKey {
		opts = append(opts, keywordStrings[UNIQUE], keywordStrings[KEY])
	}
	if ct.KeyOpt == ColKey {
		opts = append(opts, keywordStrings[KEY])
	}

	if len(opts) != 0 {
		buf.Myprintf(" %s", strings.Join(opts, " "))
	}
}

// DescribeType returns the abbreviated type information as required for
// describe table
func (ct *ColumnType) DescribeType() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%s", ct.Type)
	if ct.Length != nil && ct.Scale != nil {
		buf.Myprintf("(%v,%v)", ct.Length, ct.Scale)
	} else if ct.Length != nil {
		buf.Myprintf("(%v)", ct.Length)
	}

	opts := make([]string, 0, 16)
	if ct.Unsigned {
		opts = append(opts, keywordStrings[UNSIGNED])
	}
	if ct.Zerofill {
		opts = append(opts, keywordStrings[ZEROFILL])
	}
	if len(opts) != 0 {
		buf.Myprintf(" %s", strings.Join(opts, " "))
	}
	return buf.String()
}

// SQLType returns the sqltypes type code for the given column
func (ct *ColumnType) SQLType() querypb.Type {
	switch ct.Type {
	case keywordStrings[TINYINT]:
		if ct.Unsigned {
			return sqltypes.Uint8
		}
		return sqltypes.Int8
	case keywordStrings[SMALLINT]:
		if ct.Unsigned {
			return sqltypes.Uint16
		}
		return sqltypes.Int16
	case keywordStrings[MEDIUMINT]:
		if ct.Unsigned {
			return sqltypes.Uint24
		}
		return sqltypes.Int24
	case keywordStrings[INT]:
		fallthrough
	case keywordStrings[INTEGER]:
		if ct.Unsigned {
			return sqltypes.Uint32
		}
		return sqltypes.Int32
	case keywordStrings[BIGINT]:
		if ct.Unsigned {
			return sqltypes.Uint64
		}
		return sqltypes.Int64
	case keywordStrings[BOOL], keywordStrings[BOOLEAN]:
		return sqltypes.Uint8
	case keywordStrings[TEXT]:
		return sqltypes.Text
	case keywordStrings[TINYTEXT]:
		return sqltypes.Text
	case keywordStrings[MEDIUMTEXT]:
		return sqltypes.Text
	case keywordStrings[LONGTEXT]:
		return sqltypes.Text
	case keywordStrings[BLOB]:
		return sqltypes.Blob
	case keywordStrings[TINYBLOB]:
		return sqltypes.Blob
	case keywordStrings[MEDIUMBLOB]:
		return sqltypes.Blob
	case keywordStrings[LONGBLOB]:
		return sqltypes.Blob
	case keywordStrings[CHAR]:
		return sqltypes.Char
	case keywordStrings[VARCHAR]:
		return sqltypes.VarChar
	case keywordStrings[CHARACTER]:
		return sqltypes.VarChar
	case keywordStrings[BINARY]:
		return sqltypes.Binary
	case keywordStrings[VARBINARY]:
		return sqltypes.VarBinary
	case keywordStrings[DATE]:
		return sqltypes.Date
	case keywordStrings[TIME]:
		return sqltypes.Time
	case keywordStrings[DATETIME]:
		return sqltypes.Datetime
	case keywordStrings[TIMESTAMP]:
		return sqltypes.Timestamp
	case keywordStrings[YEAR]:
		return sqltypes.Year
	case keywordStrings[FLOAT]:
		return sqltypes.Float32
	case keywordStrings[DOUBLE]:
		return sqltypes.Float64
	case keywordStrings[DECIMAL]:
		return sqltypes.Decimal
	case keywordStrings[BIT]:
		return sqltypes.Bit
	case keywordStrings[ENUM]:
		return sqltypes.Enum
	case keywordStrings[JSON]:
		return sqltypes.TypeJSON
	case keywordStrings[GEOMETRY]:
		return sqltypes.Geometry
	case keywordStrings[POINT]:
		return sqltypes.Geometry
	case keywordStrings[LINESTRING]:
		return sqltypes.Geometry
	case keywordStrings[POLYGON]:
		return sqltypes.Geometry
	case keywordStrings[GEOMETRYCOLLECTION]:
		return sqltypes.Geometry
	case keywordStrings[MULTIPOINT]:
		return sqltypes.Geometry
	case keywordStrings[MULTILINESTRING]:
		return sqltypes.Geometry
	case keywordStrings[MULTIPOLYGON]:
		return sqltypes.Geometry
	case "character varying":
		return sqltypes.VarChar
	}
	panic("unimplemented type " + ct.Type)
}

// WalkSubtree walks the nodes of the subtree.
func (ct *ColumnType) WalkSubtree(visit Visit) error {
	return nil
}

// IndexDefinition describes an index in a CREATE TABLE statement
type IndexDefinition struct {
	Info            *IndexInfo
	Using           string
	TblName         TableName
	Columns         []*IndexColumn
	IndexOpts       []*IndexOption
	AlgorithmOption string
	LockOption      string
}

// Format formats the node.
func (idx *IndexDefinition) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v ", idx.Info)
	if idx.Using != "" {
		buf.Myprintf("using %s ", strings.ToLower(idx.Using))
	}
	if !idx.TblName.IsEmpty() {
		buf.Myprintf("on %v(", idx.TblName)
	} else {
		buf.Myprintf("(")
	}
	for i, col := range idx.Columns {
		if i != 0 {
			buf.Myprintf(", `%s`", col.Column.String())
		} else {
			buf.Myprintf("`%s`", col.Column.String())
		}
		if col.Length != nil {
			buf.Myprintf("(%v)", col.Length)
		}
		if col.Order != "" {
			buf.Myprintf(" %s", col.Order)
		}
	}
	buf.Myprintf(")")
	for _, io := range idx.IndexOpts {
		if io == nil {
			continue
		}
		buf.Myprintf(" %v", io)
	}
	if idx.AlgorithmOption != "" {
		buf.Myprintf(" algorithm %s", idx.AlgorithmOption)
	}
	if idx.LockOption != "" {
		buf.Myprintf(" lock %s", idx.LockOption)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (idx *IndexDefinition) WalkSubtree(visit Visit) error {
	if idx == nil {
		return nil
	}
	if err := Walk(visit, idx.Info, idx.TblName); err != nil {
		return err
	}
	for _, n := range idx.Columns {
		if err := Walk(visit, n.Column); err != nil {
			return err
		}
	}
	for _, n := range idx.IndexOpts {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}
	return nil
}

// IndexOption descibes the index option
type IndexOption struct {
	KeyBlockSize *SQLVal
	Using        string
	PaserName    string
	Comment      *SQLVal
}

// Format formats the node.
func (idxOpt *IndexOption) Format(buf *TrackedBuffer) {
	if idxOpt.KeyBlockSize != nil {
		buf.Myprintf("key_block_size (%v)", idxOpt.KeyBlockSize)
	} else if idxOpt.Using != "" {
		buf.Myprintf("using %s", strings.ToLower(idxOpt.Using))
	} else if idxOpt.PaserName != "" {
		buf.Myprintf("with parser %s", idxOpt.PaserName)
	} else if idxOpt.Comment != nil {
		buf.Myprintf("comment %v", idxOpt.Comment)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (idxOpt *IndexOption) WalkSubtree(visit Visit) error {
	if idxOpt == nil {
		return nil
	}

	if idxOpt.KeyBlockSize.String() != "" {
		if err := Walk(visit, idxOpt.KeyBlockSize); err != nil {
			return err
		}
	} else if idxOpt.Comment != nil {
		if err := Walk(visit, idxOpt.Comment); err != nil {
			return err
		}
	}
	return nil
}

// IndexInfo describes the name and type of an index in a CREATE TABLE statement
type IndexInfo struct {
	Type     string
	Name     TableIdent
	Primary  bool
	Spatial  bool
	Unique   bool
	Fulltext bool
}

// Format formats the node.
func (ii *IndexInfo) Format(buf *TrackedBuffer) {
	if ii.Primary || ii.Name.IsEmpty() {
		buf.Myprintf("%s", ii.Type)
	} else {
		buf.Myprintf("%s %v", ii.Type, ii.Name)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (ii *IndexInfo) WalkSubtree(visit Visit) error {
	// if err := Walk(visit, ii.Name); err != nil {
	// 	return err
	// }
	//
	return nil
}

// IndexColumn describes a column in an index definition with optional length
type IndexColumn struct {
	Column ColIdent
	Length *SQLVal
	Order  string
}

// LengthScaleOption is used for types that have an optional length
// and scale
type LengthScaleOption struct {
	Length *SQLVal
	Scale  *SQLVal
}

// ColumnKeyOption indicates whether or not the given column is defined as an
// index element and contains the type of the option
type ColumnKeyOption int

const (
	ColKeyNone ColumnKeyOption = iota
	ColKeyPrimary
	ColKeySpatialKey
	ColKeyUnique
	ColKeyUniqueKey
	ColKey
)

// ConstraintDefinition define a constraint
type ConstraintDefinition struct {
	// Name is the name of constraint
	Name   string
	Detail ConstraintDetail
	State  *ConstraintState
}

// Format formats the node.
func (c *ConstraintDefinition) Format(buf *TrackedBuffer) {
	if c == nil {
		return
	}
	format := false
	if _, ok := c.Detail.(*CheckCondition); ok {
		if FormatCheckConstraint {
			format = true
		}
	} else {
		format = true
	}
	if format {
		if c.Name != "" {
			buf.Myprintf("CONSTRAINT `%s` ", c.Name)
		}
		c.Detail.Format(buf)
		if c.State != nil {
			buf.Myprintf("%s", " ")
			c.State.Format(buf)
		}
	}
}

// WalkSubtree walks the nodes of the subtree.
func (c *ConstraintDefinition) WalkSubtree(visit Visit) error {
	return Walk(visit, c.Detail)
}

func NewConstraintDef(name string, detail ConstraintDetail, state *ConstraintState) *ConstraintDefinition {
	return &ConstraintDefinition{
		Name:   name,
		Detail: detail,
		State:  state,
	}
}

type ConstraintState struct {
	Enable   bool
	Validate bool
}

// Format formats the node.
func (cs *ConstraintState) Format(buf *TrackedBuffer) {
	if cs.Enable && cs.Validate {
		buf.Myprintf("%s %s", "ENABLE", "VALIDATE")
	} else if cs.Enable && !cs.Validate {
		buf.Myprintf("%s %s", "ENABLE", "NOVALIDATE")
	} else if !cs.Enable && cs.Validate {
		buf.Myprintf("%s %s", "DISABLE", "VALIDATE")
	} else {
		buf.Myprintf("%s %s", "DISABLE", "NOVALIDATE")
	}
}

// WalkSubtree walks the nodes of the subtree.
func (cs *ConstraintState) WalkSubtree(buf *TrackedBuffer) {
	return
}

type ConstraintStateItemType int

const (
	TypEnable ConstraintStateItemType = iota
	TypValidate
)

type ConstraintStateItem struct {
	Typ            ConstraintStateItemType
	IsEnabled      bool
	ShouldValidate bool
}

func mergeConstraintState(items []*ConstraintStateItem) *ConstraintState {
	var ret *ConstraintState = &ConstraintState{
		Enable:   true,
		Validate: true,
	}
	if len(items) == 0 {
		return nil
	} else if len(items) == 1 {
		if items[0] == nil {
			return nil
		}
		switch items[0].Typ {
		case TypEnable:
			if items[0].IsEnabled {
				ret.Enable = true
				ret.Validate = true
			} else {
				ret.Enable = false
				ret.Validate = false
			}
			return ret
		}
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		switch item.Typ {
		case TypEnable:
			ret.Enable = item.IsEnabled
		case TypValidate:
			ret.Validate = item.ShouldValidate
		}
	}
	return ret
}

type ConstraintTyp int32

const (
	TypNotNull ConstraintTyp = iota
	TypUnique
	TypPrimary
	TypForeign
	TypCheck
	TypPartition
)

func (ct ConstraintTyp) String() string {
	switch int32(ct) {
	case 0:
		return "Unknown"
	case 1:
		return "Unique"
	case 2:
		return "Primary"
	case 3:
		return "Foreign"
	case 4:
		return "Check"
	case 5:
		return "Partition"
	}
	return ""
}

// ConstraintDetail details a constraint in a CREATE TABLE statement
type ConstraintDetail interface {
	GetTyp() ConstraintTyp
	SQLNode
	iConstraintInfo()
}

func (f *ForeignKeyDetail) iConstraintInfo()   {}
func (f *PrimaryKeyDetail) iConstraintInfo()   {}
func (f *UniqueKeyDetail) iConstraintInfo()    {}
func (f *PartitionKeyDetail) iConstraintInfo() {}
func (f *CheckCondition) iConstraintInfo()     {}

func (f *ForeignKeyDetail) GetTyp() ConstraintTyp {
	return TypForeign
}

func (f *PrimaryKeyDetail) GetTyp() ConstraintTyp {
	return TypPrimary
}

func (f *UniqueKeyDetail) GetTyp() ConstraintTyp {
	return TypUnique
}

func (f *PartitionKeyDetail) GetTyp() ConstraintTyp {
	return TypPartition
}

func (f *CheckCondition) GetTyp() ConstraintTyp {
	return TypCheck
}

// PartitionKeyDetail describes a partition key in a CREATE TABLE statement
type PartitionKeyDetail struct {
	Source           ColIdent
	ReferencedTable  TableName
	ReferencedColumn ColIdent
}

var _ ConstraintDetail = &PartitionKeyDetail{}

// Format formats the node.
func (f *PartitionKeyDetail) Format(buf *TrackedBuffer) {
	buf.Myprintf("partition key(%v) references %v(%v)", f.Source, f.ReferencedTable, f.ReferencedColumn)
}

// WalkSubtree walks the nodes of the subtree.
func (f *PartitionKeyDetail) WalkSubtree(visit Visit) error {
	if err := Walk(visit, f.Source); err != nil {
		return err
	}
	if err := Walk(visit, f.ReferencedTable); err != nil {
		return err
	}
	return Walk(visit, f.ReferencedColumn)
}

// CheckCondition describes a expression constraints
type CheckCondition struct {
	CheckExpr Expr
}

// String()
func (f *CheckCondition) String() string {
	buf := NewTrackedBuffer(nil)
	f.Format(buf)
	return buf.String()
}

// IsSupported
func (f *CheckCondition) IsSupported() bool {
	switch expr := f.CheckExpr.(type) {
	case *ComparisonExpr:
		if expr.IsSimple() {
			return true
		}
		return false
	default:
		return false
	}
}

// IsOneColumn returns true if all expression constains only one column
func (f *CheckCondition) IsOneColumn() (bool, string) {
	counter := 0
	name := ""
	f.WalkSubtree(func(node SQLNode) (kontinue bool, err error) {
		if cn, ok := node.(*ColName); ok {
			counter++
			name = cn.Name.String()
			return false, nil
		}
		return true, nil
	})
	return counter == 1, name
}

// Format formats the node.
func (f *CheckCondition) Format(buf *TrackedBuffer) {
	if f != nil {
		buf.Myprintf("CHECK (%v)", f.CheckExpr)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (f *CheckCondition) WalkSubtree(visit Visit) error {
	return f.CheckExpr.WalkSubtree(visit)
}

// ReferenceAction indicates the action takes by a referential constraint e.g.
// the `CASCADE` in a `FOREIGN KEY .. ON DELETE CASCADE` table definition.
type ReferenceAction int

// These map to the SQL-defined reference actions.
// See https://dev.mysql.com/doc/refman/8.0/en/create-table-foreign-keys.html#foreign-keys-referential-actions
const (
	// DefaultAction indicates no action was explicitly specified.
	DefaultAction ReferenceAction = iota
	Restrict
	Cascade
	NoAction
	SetNull
	SetDefault
)

// WalkSubtree walks the nodes of the subtree.
func (ra ReferenceAction) WalkSubtree(visit Visit) error { return nil }

// Format formats the node.
func (ra ReferenceAction) Format(buf *TrackedBuffer) {
	switch ra {
	case Restrict:
		buf.WriteString("RESTRICT")
	case Cascade:
		buf.WriteString("CASCADE")
	case NoAction:
		buf.WriteString("NO ACTION")
	case SetNull:
		buf.WriteString("SET NULL")
	case SetDefault:
		buf.WriteString("SET DEFAULT")
	}
}

// OnUpdateDelete is the action of reference
type OnUpdateDelete struct {
	OnDelete ReferenceAction
	OnUpdate ReferenceAction
}

// Format formats the node.
func (oud OnUpdateDelete) Format(buf *TrackedBuffer) {
	if oud.OnDelete != DefaultAction {
		buf.Myprintf(" ON DELETE %v", oud.OnDelete)
	}
	if oud.OnUpdate != DefaultAction {
		buf.Myprintf(" ON UPDATE %v", oud.OnUpdate)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (oud OnUpdateDelete) WalkSubtree(visit Visit) error {
	return nil
}

// Reference describes the relationships among tables
type Reference struct {
	ReferencedTable   TableName
	ReferencedColumns Columns
	Actions           OnUpdateDelete
}

// Format formats the node.
func (r Reference) Format(buf *TrackedBuffer) {
	buf.Myprintf("REFERENCES %v %v%v", r.ReferencedTable, r.ReferencedColumns, r.Actions)
}

// WalkSubtree walks the nodes of the subtree.
func (r Reference) WalkSubtree(visit Visit) error {
	if err := Walk(visit, r.ReferencedTable); err != nil {
		return err
	}
	return Walk(visit, r.ReferencedColumns)
}

// ForeignKeyDetail describes a foreign key in a CREATE TABLE statement
type ForeignKeyDetail struct {
	IndexName     string
	Source        Columns
	ReferenceInfo Reference
}

// Format formats the node.
func (f *ForeignKeyDetail) Format(buf *TrackedBuffer) {
	if f == nil {
		return
	}
	buf.Myprintf("FOREIGN KEY %v %v", f.Source, f.ReferenceInfo)
}

// WalkSubtree walks the nodes of the subtree.
func (f *ForeignKeyDetail) WalkSubtree(visit Visit) error {
	if err := Walk(visit, f.Source); err != nil {
		return err
	}
	return Walk(visit, f.ReferenceInfo)
}

// PrimaryKeyDetail describes a foreign key in a CREATE TABLE statement
type PrimaryKeyDetail struct {
}

// Format formats the node.
func (f *PrimaryKeyDetail) Format(buf *TrackedBuffer) {
	return
}

// WalkSubtree walks the nodes of the subtree.
func (f *PrimaryKeyDetail) WalkSubtree(visit Visit) error {
	return nil
}

// UniqueKeyDetail describes a foreign key in a CREATE TABLE statement
type UniqueKeyDetail struct {
}

// Format formats the node.
func (f *UniqueKeyDetail) Format(buf *TrackedBuffer) {
	return
}

// WalkSubtree walks the nodes of the subtree.
func (f *UniqueKeyDetail) WalkSubtree(visit Visit) error {
	return nil
}

// Show represents a show statement.
type Show struct {
	Type       string
	FullShow   bool
	DBSpec     *DatabaseSpec
	TbFromSpec *TableFromSpec
	Principal  *Principal
	Condition  ShowCondition
	IsGlobal   bool
}

// DatabaseSpec represents from database spec in a show
type DatabaseSpec struct {
	// DbName is the source database name
	DBName TableIdent
	// keyspace information must be added manually
	keyspace string
}

// NewDatabaseSpec return a DatabaseSpec
func NewDatabaseSpec(name TableIdent) *DatabaseSpec {
	return &DatabaseSpec{
		DBName: name,
	}
}

// AddKsInfo add correct keyspace information
func (node *DatabaseSpec) AddKsInfo(ks string) {
	if node == nil {
		return
	}
	node.keyspace = ks
}

// Format formats the node.
func (node *DatabaseSpec) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	buf.Myprintf(" from %v", node.DBName)
}

// WalkSubtree walks the nodes of the subtree.
func (node *DatabaseSpec) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(visit, node.DBName)
}

// TableFromSpec represents table from spec in a show
type TableFromSpec struct {
	// TbName is the source table name
	TbName   TableName
	keyspace string
}

// NewTableFromSpec return a TableFromSpec
func NewTableFromSpec(name TableName) *TableFromSpec {
	return &TableFromSpec{
		TbName: name,
	}
}

// AddKsInfo add correct keyspace information
func (node *TableFromSpec) AddKsInfo(ks string) {
	if node == nil {
		return
	}
	node.keyspace = ks
}

// Format formats the node.
func (node *TableFromSpec) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	buf.Myprintf(" from %v", node.TbName)
}

// WalkSubtree walks the nodes of the subtree.
func (node *TableFromSpec) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(visit, node.TbName)
}

// ShowCondition is the common interface for
// conditions in show statement
type ShowCondition interface {
	iShowCondition()
}

func (*ShowLikeExpr) iShowCondition() {}
func (*Where) iShowCondition()        {}

// ShowLikeExpr represents like spec in a show statement
type ShowLikeExpr struct {
	// LikeStr is the like pattern string
	LikeStr string
}

// Format formats the node.
func (node *ShowLikeExpr) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	buf.Myprintf(" like '%s'", node.LikeStr)
}

// WalkSubtree walks the nodes of the subtree.
func (node *ShowLikeExpr) WalkSubtree(visit Visit) error {
	return nil
}

// The following constants represent SHOW statements.
const (
	ShowSchemasStr      = "schemas"
	ShowDatabasesStr    = "databases"
	ShowTablesStr       = "tables"
	ShowColumnsStr      = "columns"
	ShowIndexesStr      = "indexes"
	ShowTriggersStr     = "triggers"
	ShowGrantsStr       = "grants"
	ShowPrivilegesStr   = "privileges"
	ShowVariablesStr    = "variables"
	ShowProcesslistStr  = "processlist"
	ShowEngineStr       = "engine"
	ShowEnginesStr      = "engines"
	ShowErrorsStr       = "errors"
	ShowWarningsStr     = "warnings"
	ShowCharacterSetStr = "character set"
	ShowCollation       = "collation"
	ShowPartitionsStr   = "partitions"

	ShowTableStatusStr       = "table status"
	ShowFunctionStatusStr    = "function status"
	ShowProcedureStatusStr   = "procedure status"
	ShowPackageStatusStr     = "package status"
	ShowPackageBodyStatusStr = "package body status"

	ShowCreateTableStr       = "create table"
	ShowCreateViewStr        = "create view"
	ShowCreateSchemaStr      = "create schema"
	ShowCreateDatabaseStr    = "create database"
	ShowCreateProcedureStr   = "create procedure"
	ShowCreateFunctionStr    = "create function"
	ShowCreateTriggerStr     = "create trigger"
	ShowCreatePackageStr     = "create package"
	ShowCreatePackageBodyStr = "create package body"

	ShowKeyspacesStr     = "kundb_keyspaces"
	ShowShardsStr        = "kundb_shards"
	ShowSessionShardsStr = "session shards"
	ShowVindexesStr      = "kundb_vindexes"
	ShowVSchemaTablesStr = "vschema_tables"
	ShowRangeInfoStr     = "kundb_range_info"
	ShowKunCheckStr      = "kundb_checks"
	ShowUnsupportedStr   = "unsupported"
)

// AddKsInfo add correct keyspace information
func (node *Show) AddKsInfo(ks string) {
	if node == nil {
		return
	}
	node.DBSpec.AddKsInfo(ks)
	node.TbFromSpec.AddKsInfo(ks)
}

// Format formats the node.
func (node *Show) Format(buf *TrackedBuffer) {
	full := ""
	if node.FullShow {
		full = " full"
	}
	switch node.Type {
	case ShowTablesStr:
		if node.Condition != nil {
			buf.Myprintf("show%s tables%v%v", full, node.DBSpec, node.Condition)
		} else {
			buf.Myprintf("show%s tables%v", full, node.DBSpec)
		}
	case ShowColumnsStr:
		if node.Condition != nil {
			buf.Myprintf("show%s columns%v%v%v", full, node.TbFromSpec, node.DBSpec, node.Condition)
		} else {
			buf.Myprintf("show%s columns%v%v", full, node.TbFromSpec, node.DBSpec)
		}
	case ShowIndexesStr:
		if node.Condition != nil {
			buf.Myprintf("show indexes%v%v%v", node.TbFromSpec, node.DBSpec, node.Condition)
		} else {
			buf.Myprintf("show indexes%v%v", node.TbFromSpec, node.DBSpec)
		}
	case ShowTriggersStr:
		if node.Condition != nil {
			buf.Myprintf("show triggers%v%v", node.DBSpec, node.Condition)
		} else {
			buf.Myprintf("show triggers%v", node.DBSpec)
		}
	case ShowTableStatusStr:
		if node.Condition != nil {
			buf.Myprintf("show table status%v%v", node.DBSpec, node.Condition)
		} else {
			buf.Myprintf("show table status%v", node.DBSpec)
		}
	case ShowFunctionStatusStr:
		if node.Condition != nil {
			buf.Myprintf("show function status%v", node.Condition)
		} else {
			buf.Myprintf("show function status")
		}
	case ShowProcedureStatusStr:
		if node.Condition != nil {
			buf.Myprintf("show procedure status%v", node.Condition)
		} else {
			buf.Myprintf("show procedure status")
		}
	case ShowCreateDatabaseStr, ShowCreateSchemaStr:
		buf.Myprintf("show create database %v", node.DBSpec.DBName)
	case ShowCreateTableStr:
		buf.Myprintf("show create table %v", node.TbFromSpec.TbName)
	case ShowCreateViewStr:
		buf.Myprintf("show create view %v", node.TbFromSpec.TbName)
	case ShowCreateProcedureStr:
		buf.Myprintf("show create procedure %v", node.TbFromSpec.TbName)
	case ShowCreateFunctionStr:
		buf.Myprintf("show create function %v", node.TbFromSpec.TbName)
	case ShowCreateTriggerStr:
		buf.Myprintf("show create trigger %v", node.TbFromSpec.TbName)
	case ShowGrantsStr:
		if node.Principal != nil {
			buf.Myprintf("show grants for %v", node.Principal)
		} else {
			buf.Myprintf("show grants")
		}
	default:
		buf.Myprintf("show %s", node.Type)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *Show) WalkSubtree(visit Visit) error {
	return nil
}

// GetIndexShowSQL returns a sql to DDL with partition option
func (node *Show) GetIndexShowSQL() (sql string, err error) {
	switch node.Type {
	case ShowIndexesStr:
		{
			return strings.ToLower(String(node)), nil
		}
	default:
		return "", fmt.Errorf("GetIndexShowSQL: action %v not supported for index", node.Type)
	}
}

// CheckInDB checks the sql is excuted in a database
// for example: show table status may return "No database selected" error if not in a database
func (node *Show) CheckInDB(curDatabase string) string {
	if node.TbFromSpec != nil && !node.TbFromSpec.TbName.Qualifier.IsEmpty() {
		return node.TbFromSpec.TbName.Qualifier.String()
	}
	if node.DBSpec != nil && !node.DBSpec.DBName.IsEmpty() {
		return node.DBSpec.DBName.String()
	}
	if curDatabase != "" {
		return curDatabase
	}
	return ""
}

// Use represents a use statement.
type Use struct {
	DBName TableIdent
}

// Format formats the node.
func (node *Use) Format(buf *TrackedBuffer) {
	buf.Myprintf("use %v", node.DBName)
}

// WalkSubtree walks the nodes of the subtree.
func (node *Use) WalkSubtree(visit Visit) error {
	return Walk(visit, node.DBName)
}

// Begin represents a Begin statement.
type Begin struct{}

// Format formats the node.
func (node *Begin) Format(buf *TrackedBuffer) {
	buf.WriteString("begin")
}

// WalkSubtree walks the nodes of the subtree.
func (node *Begin) WalkSubtree(visit Visit) error {
	return nil
}

// Commit represents a Commit statement.
type Commit struct{}

// Format formats the node.
func (node *Commit) Format(buf *TrackedBuffer) {
	buf.WriteString("commit")
}

// WalkSubtree walks the nodes of the subtree.
func (node *Commit) WalkSubtree(visit Visit) error {
	return nil
}

// Rollback represents a Rollback statement.
type Rollback struct{}

// Format formats the node.
func (node *Rollback) Format(buf *TrackedBuffer) {
	buf.WriteString("rollback")
}

// WalkSubtree walks the nodes of the subtree.
func (node *Rollback) WalkSubtree(visit Visit) error {
	return nil
}

// Call represents a call some_procedure statement
type Call struct {
	ProcName ProcOrFuncName
}

// Format formats the node.
func (call *Call) Format(buf *TrackedBuffer) {
	buf.Myprintf("call %v", call.ProcName)
}

// WalkSubtree walks the nodes of the subtree.
func (call *Call) WalkSubtree(visit Visit) error {
	return Walk(visit, call.ProcName)
}

type otherReadType int

// other read types
const (
	// UndType is the default otherRead type
	UndType otherReadType = iota
	TableOrView
	Others
)

// OtherRead represents a DESCRIBE, or EXPLAIN statement.
// It should be used only as an indicator. It does not contain
// the full AST for the statement.
type OtherRead struct {
	OtherReadType otherReadType
	Object        TableName
}

// Format formats the node.
func (node *OtherRead) Format(buf *TrackedBuffer) {
	buf.WriteString("otherread")
}

// WalkSubtree walks the nodes of the subtree.
func (node *OtherRead) WalkSubtree(visit Visit) error {
	return nil
}

// Other admin strings.
const (
	AnalyzeStr  = "analyze"
	OptimizeStr = "optimize"
	// Explain formats
	TreeStr        = "tree"
	JSONStr        = "json"
	TraditionalStr = "traditional"
)

// OtherAdmin represents a misc statement that relies on ADMIN privileges,
// such as REPAIR, OPTIMIZE, or TRUNCATE statement.
// It should be used only as an indicator. It does not contain
// the full AST for the statement.
type OtherAdmin struct {
	Action   string
	AdminOpt string
	Table    TableName
}

// Format formats the node.
func (node *OtherAdmin) Format(buf *TrackedBuffer) {
	if !node.Table.IsEmpty() {
		if node.AdminOpt != "" {
			buf.Myprintf("%s %s table %v", node.Action, node.AdminOpt, node.Table)
		} else {
			buf.Myprintf("%s table %v", node.Action, node.Table)
		}
	} else {
		buf.WriteString("otheradmin")
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *OtherAdmin) WalkSubtree(visit Visit) error {
	if !node.Table.IsEmpty() {
		if err := Walk(visit, node.Table); err != nil {
			return err
		}
	}
	return nil
}

// Comments represents a list of comments.
type Comments [][]byte

// Format formats the node.
func (node Comments) Format(buf *TrackedBuffer) {
	for _, c := range node {
		buf.Myprintf("%s ", c)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node Comments) WalkSubtree(visit Visit) error {
	return nil
}

// Constains returns true is comments constains target string
func (node Comments) Contains(target string) bool {
	for _, c := range node {
		cStr := string(c)
		if strings.Contains(cStr, target) {
			return true
		}
	}
	return false
}

// SelectExprs represents SELECT expressions.
type SelectExprs []SelectExpr

// Format formats the node.
func (node SelectExprs) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Myprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node SelectExprs) WalkSubtree(visit Visit) error {
	for _, n := range node {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}
	return nil
}

// SelectExpr represents a SELECT expression.
type SelectExpr interface {
	iSelectExpr()
	SQLNode
}

func (*StarExpr) iSelectExpr()    {}
func (*AliasedExpr) iSelectExpr() {}
func (Nextval) iSelectExpr()      {}

// StarExpr defines a '*' or 'table.*' expression.
type StarExpr struct {
	TableName TableName
}

// Format formats the node.
func (node *StarExpr) Format(buf *TrackedBuffer) {
	if !node.TableName.IsEmpty() {
		buf.Myprintf("%v.", node.TableName)
	}
	buf.Myprintf("*")
}

// WalkSubtree walks the nodes of the subtree.
func (node *StarExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.TableName,
	)
}

// AliasedVarExpr defines an aliased SELECT expression.
type AliasedVarExpr struct {
	SExpr string
	AsOpt bool
	As    Expr
}

// AliasedExpr defines an aliased SELECT expression.
type AliasedExpr struct {
	Expr  Expr
	AsOpt bool
	As    Expr
}

// Format formats the node.
func (node *AliasedExpr) Format(buf *TrackedBuffer) {
	hasAlias := false
	if node.AsOpt {
		hasAlias = true
	}
	buf.Myprintf("%v", node.Expr)
	if hasAlias {
		buf.Myprintf(" as %v", node.As)
		return
	}
	switch node := node.Expr.(type) {
	case *RownumExpr:
		buf.rownumIndex++
		buf.Myprintf(" as `rownum`")
	case *FuncExpr:
		node.AddOracleAlias(buf)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *AliasedExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Expr,
		node.As,
	)
}

// Nextval defines the NEXT VALUE expression.
type Nextval struct {
	Expr Expr
}

// Format formats the node.
func (node Nextval) Format(buf *TrackedBuffer) {
	buf.Myprintf("next value")
}

// WalkSubtree walks the nodes of the subtree.
func (node Nextval) WalkSubtree(visit Visit) error {
	return Walk(visit, node.Expr)
}

// RownumExpr defines an rownum expression.
type RownumExpr struct {
}

// Format formats the node.
func (node *RownumExpr) Format(buf *TrackedBuffer) {
	rownumIndex := buf.rownumIndex
	buf.Myprintf(rnSelectItemFormat, rownumIndex, rownumIndex)
}

// WalkSubtree walks the nodes of the subtree.
func (node *RownumExpr) WalkSubtree(visit Visit) error {
	return nil
}

// Columns represents an insert column list.
type Columns []ColIdent

// Format formats the node.
func (node Columns) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	prefix := "("
	for _, n := range node {
		buf.Myprintf("%s%v", prefix, n)
		prefix = ", "
	}
	buf.WriteString(")")
}

// WalkSubtree walks the nodes of the subtree.
func (node Columns) WalkSubtree(visit Visit) error {
	for _, n := range node {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}
	return nil
}

// FindColumn finds a column in the column list, returning
// the index if it exists or -1 otherwise
func (node Columns) FindColumn(col ColIdent) int {
	for i, colName := range node {
		if colName.Equal(col) {
			return i
		}
	}
	return -1
}

// TableExprs represents a list of table expressions.
type TableExprs []TableExpr

// Format formats the node.
func (node TableExprs) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Myprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node TableExprs) WalkSubtree(visit Visit) error {
	for _, n := range node {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}
	return nil
}

// TableExpr represents a table expression.
type TableExpr interface {
	iTableExpr()
	GetAllTables() ([]string, error)
	SQLNode
}

func (*AliasedTableExpr) iTableExpr() {}
func (*ParenTableExpr) iTableExpr()   {}
func (*JoinTableExpr) iTableExpr()    {}

// AliasedTableExpr represents a table expression
// coupled with an optional alias or index hint.
// If As is empty, no alias was used.
type AliasedTableExpr struct {
	Expr             SimpleTableExpr
	Partition        TableIdent
	As               TableIdent
	Hints            *IndexHints
	ScanMode         *ScanMode
	QualifierChanged bool
}

// Format formats the node.
func (node *AliasedTableExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v", node.Expr)
	if !node.Partition.IsEmpty() {
		buf.Myprintf(" (%v)", node.Partition)
	}
	if !node.As.IsEmpty() {
		buf.Myprintf(" as %v", node.As)
	}
	if node.Hints != nil {
		// Hint node provides the space padding.
		buf.Myprintf("%v", node.Hints)
	}
	if node.ScanMode != nil {
		buf.Myprintf("%v", node.ScanMode)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *AliasedTableExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Expr,
		node.As,
		node.Hints,
		node.ScanMode,
	)
}

// GetAllTables return all the tables in the expression
func (node *AliasedTableExpr) GetAllTables() ([]string, error) {
	switch node2 := node.Expr.(type) {
	case TableName:
		table := node2.Name.String()
		var tableName []string
		tableName = append(tableName, table)
		return tableName, nil
	case *Subquery:
		return nil, errors.New(errors.ErrNotSupport, "not support subquery in get all tables")
	}
	return nil, fmt.Errorf("should not be here, expr is %v", node.Expr)
}

// SimpleTableExpr represents a simple table expression.
type SimpleTableExpr interface {
	iSimpleTableExpr()
	SQLNode
}

func (TableName) iSimpleTableExpr() {}
func (*Subquery) iSimpleTableExpr() {}

// TableNames is a list of TableName.
type TableNames []TableName

// Format formats the node.
func (node TableNames) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Myprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node TableNames) WalkSubtree(visit Visit) error {
	for _, n := range node {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}
	return nil
}

// TableName represents a table  name.
// Qualifier, if specified, represents a database or keyspace.
// TableName is a value struct whose fields are case sensitive.
// This means two TableName vars can be compared for equality
// and a TableName can also be used as key in a map.
type TableName struct {
	Name, Qualifier TableIdent
}

func NewTableName(qualifier, name string) TableName {
	return TableName{
		Name:      NewTableIdent(name),
		Qualifier: NewTableIdent(qualifier),
	}
}

// Format formats the node.
func (node TableName) Format(buf *TrackedBuffer) {
	if node.IsEmpty() {
		return
	}
	if !node.Qualifier.IsEmpty() {
		buf.Myprintf("%v.", node.Qualifier)
	}
	buf.Myprintf("%v", node.Name)
}

// WalkSubtree walks the nodes of the subtree.
func (node TableName) WalkSubtree(visit Visit) error {
	return Walk(
		visit,
		node.Name,
		node.Qualifier,
	)
}

// IsEmpty returns true if TableName is nil or empty.
func (node TableName) IsEmpty() bool {
	// If Name is empty, Qualifer is also empty.
	return node.Name.IsEmpty()
}

// ToViewName returns a TableName acceptable for use as a VIEW. VIEW names are
// always lowercase, so ToViewName lowercasese the name. Databases are case-sensitive
// so Qualifier is left untouched.
func (node TableName) ToViewName() TableName {
	return TableName{
		Qualifier: node.Qualifier,
		Name:      NewTableIdent(strings.ToLower(node.Name.v)),
	}
}

// ProcOrFuncName represents a Procedure name or a Function name.
// Procedure or functions may store in pacakges, and may be called like
// CALL [schema].[package].[procedure]
type ProcOrFuncName struct {
	Name, Package, Qualifier TableIdent
}

// Format formats the node.
func (node ProcOrFuncName) Format(buf *TrackedBuffer) {
	if node.IsEmpty() {
		return
	}
	if !node.Qualifier.IsEmpty() {
		buf.Myprintf("%v.", node.Qualifier)
	}
	if !node.Package.IsEmpty() {
		buf.Myprintf("%v.", node.Package)
	}
	buf.Myprintf("%v", node.Name)
}

// WalkSubtree walks the nodes of the subtree.
func (node ProcOrFuncName) WalkSubtree(visit Visit) error {
	return Walk(
		visit,
		node.Name,
		node.Package,
		node.Qualifier,
	)
}

// IsEmpty returns true if TableName is nil or empty.
func (node ProcOrFuncName) IsEmpty() bool {
	// If Name is empty, Qualifer is also empty.
	return node.Name.IsEmpty()
}

// ParenTableExpr represents a parenthesized list of TableExpr.
type ParenTableExpr struct {
	Exprs TableExprs
}

// Format formats the node.
func (node *ParenTableExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("(%v)", node.Exprs)
}

// WalkSubtree walks the nodes of the subtree.
func (node *ParenTableExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Exprs,
	)
}

// GetAllTables return all the tables in the expression
func (node *ParenTableExpr) GetAllTables() ([]string, error) {
	return nil, errors.New(errors.ErrNotSupport, "not supported now")
}

// JoinTableExpr represents a TableExpr that's a JOIN operation.
type JoinTableExpr struct {
	LeftExpr  TableExpr
	Join      string
	RightExpr TableExpr
	On        Expr
	Patchwork BoolVal
}

// JoinTableExpr.Join
const (
	JoinStr             = "join"
	StraightJoinStr     = "straight_join"
	LeftJoinStr         = "left join"
	RightJoinStr        = "right join"
	NaturalJoinStr      = "natural join"
	NaturalLeftJoinStr  = "natural left join"
	NaturalRightJoinStr = "natural right join"
)

// Format formats the node.
func (node *JoinTableExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v %s %v", node.LeftExpr, node.Join, node.RightExpr)
	if node.On != nil {
		buf.Myprintf(" on %v", node.On)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *JoinTableExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.LeftExpr,
		node.RightExpr,
		node.On,
	)
}

// GetAllTables return all the tables in the expression
func (node *JoinTableExpr) GetAllTables() ([]string, error) {
	return nil, errors.New(errors.ErrNotSupport, "not supported now")
}

// ScanMode represents the scan mode hint
type ScanMode struct {
	Mode *SQLVal
}

// Format formats the node.
func (node *ScanMode) Format(buf *TrackedBuffer) {
	buf.Myprintf(" FETCH MODE %v", node.Mode)
}

// WalkSubtree walks the nodes of the subtree.
func (node *ScanMode) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(visit, node.Mode)
}

// IndexHints represents a list of index hints.
type IndexHints struct {
	Type    string
	Indexes []ColumnDetail
}

// Index hints.
const (
	UseStr    = "use "
	IgnoreStr = "ignore "
	ForceStr  = "force "
)

// Format formats the node.
func (node *IndexHints) Format(buf *TrackedBuffer) {
	buf.Myprintf(" %sindex ", node.Type)
	prefix := "("
	for _, n := range node.Indexes {
		buf.Myprintf("%s%v", prefix, n)
		prefix = ", "
	}
	buf.Myprintf(")")
}

// WalkSubtree walks the nodes of the subtree.
func (node *IndexHints) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	for _, n := range node.Indexes {
		if err := Walk(visit, n.ColName); err != nil {
			return err
		}
	}
	return nil
}

// Where represents a WHERE or HAVING clause.
type Where struct {
	Type TypClause
	Expr Expr
}

type TypClause int

// Where.Type
const (
	TypWhere TypClause = iota
	TypHaving
)

func (tc TypClause) String() string {
	switch tc {
	case TypWhere:
		return "where"
	case TypHaving:
		return "having"
	}
	return ""
}

// NewWhere creates a WHERE or HAVING clause out
// of a Expr. If the expression is nil, it returns nil.
func NewWhere(typ TypClause, expr Expr) *Where {
	if expr == nil {
		return nil
	}
	return &Where{Type: typ, Expr: expr}
}

// RemoveRownum remove the specifical expr that contains rownum
// out parameters: new Where after removing rownum, removed exprs, error
func (node *Where) RemoveRownum() (newWhere *Where, removed Exprs, err error) {
	if node == nil {
		return node, nil, fmt.Errorf("null pointer, where is null")
	}
	if !NodeHasRownum(node) {
		return node, nil, nil
	}
	switch expr := node.Expr.(type) {
	case *AndExpr:
		// a and (b or rownum < xx) will generate [a, b or rownum < xx], not support
		filters := SplitAndExpression(nil, expr)
		var afterRemove []Expr
		afterRemove, removed, err = RemoveRownum(filters)
		if err != nil {
			return node, nil, err
		}
		newWhere = NewWhere(node.Type, RecombinedExpression(afterRemove))
	case *OrExpr:
		// not support
		return node, nil, fmt.Errorf("rownum in or-expr is not suppport")
	case *ComparisonExpr:
		removed = append(removed, expr)
	case *RangeCond:
		removed = append(removed, expr)
	}
	return
}

// Format formats the node.
func (node *Where) Format(buf *TrackedBuffer) {
	if node == nil || node.Expr == nil {
		return
	}
	buf.Myprintf(" %s %v", node.Type.String(), node.Expr)
}

// WalkSubtree walks the nodes of the subtree.
func (node *Where) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Expr,
	)
}

type ExprTyp int32

const (
	TypAssign ExprTyp = iota
	TypAnd
	TypOr
	TypNot
	TypParent
	TypComparison
	TypRangeCond
	TypIs
	TypExists
	TypSQLVal
	TypNullVal
	TypBoolVal
	TypColName
	TypValTuple
	TypSubQuery
	TypListArg
	TypBinary
	TypUnary
	TypInterval
	TypCollate
	TypFunc
	TypCase
	TypValuesFunc
	TypConvert
	TypConvertUsing
	TypMatch
	TypGroupConcat
	TypConcat
	TypDefault
	TypRownum
)

// Expr represents an expression.
type Expr interface {
	SQLNode
	GetType() ExprTyp
	String() string
}

func ParseInt(expr Expr) (int, error) {
	sqlVal, ok := expr.(*SQLVal)
	if !ok {
		return 0, errors.New(errors.ErrNotSupport, fmt.Sprintf("not sqlval, expr is '%s'", expr.String()))
	}
	if sqlVal.Type != IntVal {
		return 0, errors.New(errors.ErrNotSupport, fmt.Sprintf("not int typ, expr is '%s'", expr.String()))
	}
	ans, err := strconv.Atoi(string(sqlVal.Val))
	if err != nil {
		return 0, err
	}
	return ans, nil
}

func IsComplicatedRownum(expr Expr) bool {
	if !NodeHasRownum(expr) {
		return false
	}
	switch expr := expr.(type) {
	case *ComparisonExpr:
		if expr.IsLike() || expr.IsRegexp() || expr.Operator == JSONExtractOp || expr.Operator == JSONUnquoteExtractOp || expr.Operator == NullSafeEqualStr {
			return true
		}
		l, r := NodeIsRownum(expr.Left), NodeIsRownum(expr.Right)
		if expr.IsIN() {
			if !l {
				return true
			}
			if _, ok := expr.Right.(ValTuple); !ok {
				return true
			}
			return false
		}
		if !l && !r {
			return true
		} else if l && !NodeIsIntVal(expr.Right) {
			return true
		} else if r && !NodeIsIntVal(expr.Left) {
			return true
		} else if l && r {
			// magic: rownum op rownum
			return true
		}
		return false
	case *RangeCond:
		if !NodeIsRownum(expr.Left) {
			return true
		}
		if !NodeIsIntVal(expr.From) || !NodeIsIntVal(expr.To) {
			return true
		}
		return false
	case *OrExpr:
		return true
	case *AndExpr:
		filters := SplitAndExpression(nil, expr)
		for _, filter := range filters {
			complicate := IsComplicatedRownum(filter)
			if complicate {
				return true
			}
		}
		return false
	case *ParenExpr:
		return IsComplicatedRownum(expr.Expr)
	}
	return true
}

func (*AssignExpr) GetType() ExprTyp       { return TypAssign }
func (*AndExpr) GetType() ExprTyp          { return TypAnd }
func (*OrExpr) GetType() ExprTyp           { return TypOr }
func (*NotExpr) GetType() ExprTyp          { return TypNot }
func (*ParenExpr) GetType() ExprTyp        { return TypParent }
func (*ComparisonExpr) GetType() ExprTyp   { return TypComparison }
func (*RangeCond) GetType() ExprTyp        { return TypRangeCond }
func (*IsExpr) GetType() ExprTyp           { return TypIs }
func (*ExistsExpr) GetType() ExprTyp       { return TypExists }
func (*SQLVal) GetType() ExprTyp           { return TypSQLVal }
func (*NullVal) GetType() ExprTyp          { return TypNullVal }
func (BoolVal) GetType() ExprTyp           { return TypBoolVal }
func (*ColName) GetType() ExprTyp          { return TypColName }
func (ValTuple) GetType() ExprTyp          { return TypValTuple }
func (*Subquery) GetType() ExprTyp         { return TypSubQuery }
func (ListArg) GetType() ExprTyp           { return TypListArg }
func (*BinaryExpr) GetType() ExprTyp       { return TypBinary }
func (*UnaryExpr) GetType() ExprTyp        { return TypUnary }
func (*IntervalExpr) GetType() ExprTyp     { return TypInterval }
func (*CollateExpr) GetType() ExprTyp      { return TypCollate }
func (*FuncExpr) GetType() ExprTyp         { return TypFunc }
func (*CaseExpr) GetType() ExprTyp         { return TypCase }
func (*ValuesFuncExpr) GetType() ExprTyp   { return TypValuesFunc }
func (*ConvertExpr) GetType() ExprTyp      { return TypConvert }
func (*ConvertUsingExpr) GetType() ExprTyp { return TypConvertUsing }
func (*MatchExpr) GetType() ExprTyp        { return TypMatch }
func (*GroupConcatExpr) GetType() ExprTyp  { return TypGroupConcat }
func (*ConcatExpr) GetType() ExprTyp       { return TypConcat }
func (*Default) GetType() ExprTyp          { return TypDefault }
func (*RownumExpr) GetType() ExprTyp       { return TypRownum }

func (node *AssignExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *AndExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *OrExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *NotExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *ParenExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *ComparisonExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *RangeCond) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *IsExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *ExistsExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *SQLVal) String() string {
	buf := NewTrackedBuffer(nil)
	switch node.Type {
	case StrVal, IntVal, FloatVal, HexNum, HexVal, BitVal:
		buf.WriteString(string(node.Val))
	case ValArg:
		buf.WriteArg(string(node.Val))
	default:
		panic("unexpected")
	}
	return buf.String()
}

func (node *NullVal) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node BoolVal) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *ColName) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node ValTuple) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *Subquery) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node ListArg) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *BinaryExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *UnaryExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *IntervalExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *CollateExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *FuncExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *CaseExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *ValuesFuncExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *ConvertExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *ConvertUsingExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *MatchExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *GroupConcatExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *ConcatExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *Default) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

func (node *RownumExpr) String() string {
	buf := NewTrackedBuffer(nil)
	buf.Myprintf("%v", node)
	return buf.String()
}

// Exprs represents a list of value expressions.
// It's not a valid expression because it's not parenthesized.
type Exprs []Expr

// Format formats the node.
func (node Exprs) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Myprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node Exprs) WalkSubtree(visit Visit) error {
	for _, n := range node {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}
	return nil
}

// AssignExpr represents an ASSIGN expression.
type AssignExpr struct {
	Left, Right Expr
}

// Format formats the node.
func (node *AssignExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v := %v", node.Left, node.Right)
}

// WalkSubtree walks the nodes of the subtree.
func (node *AssignExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Left,
		node.Right,
	)
}

// AndExpr represents an AND expression.
type AndExpr struct {
	Left, Right Expr
}

// Format formats the node.
func (node *AndExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v and %v", node.Left, node.Right)
}

// WalkSubtree walks the nodes of the subtree.
func (node *AndExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Left,
		node.Right,
	)
}

// OrExpr represents an OR expression.
type OrExpr struct {
	Left, Right Expr
}

// Format formats the node.
func (node *OrExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v or %v", node.Left, node.Right)
}

// WalkSubtree walks the nodes of the subtree.
func (node *OrExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Left,
		node.Right,
	)
}

// NotExpr represents a NOT expression.
type NotExpr struct {
	Expr Expr
}

// Format formats the node.
func (node *NotExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("not %v", node.Expr)
}

// WalkSubtree walks the nodes of the subtree.
func (node *NotExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Expr,
	)
}

// ParenExpr represents a parenthesized boolean expression.
type ParenExpr struct {
	Expr Expr
}

// Format formats the node.
func (node *ParenExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("(%v)", node.Expr)
}

// WalkSubtree walks the nodes of the subtree.
func (node *ParenExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Expr,
	)
}

// ComparisonExpr represents a two-value comparison expression.
type ComparisonExpr struct {
	Operator    string
	Left, Right Expr
	Escape      Expr
}

// ComparisonExpr.Operator
const (
	EqualStr             = "="
	LessThanStr          = "<"
	GreaterThanStr       = ">"
	LessEqualStr         = "<="
	GreaterEqualStr      = ">="
	NotEqualStr          = "!="
	NullSafeEqualStr     = "<=>"
	InStr                = "in"
	NotInStr             = "not in"
	LikeStr              = "like"
	NotLikeStr           = "not like"
	RegexpStr            = "regexp"
	NotRegexpStr         = "not regexp"
	JSONExtractOp        = "->"
	JSONUnquoteExtractOp = "->>"
)

func (node *ComparisonExpr) IsSimple() bool {
	if node.Operator == EqualStr || node.Operator == LessThanStr || node.Operator == GreaterThanStr ||
		node.Operator == LessEqualStr || node.Operator == GreaterEqualStr || node.Operator == NotEqualStr {
		if node.Left.GetType() == TypSQLVal && node.Right.GetType() == TypColName {
			sqlVal, _ := node.Left.(*SQLVal)
			if sqlVal.Type != IntVal {
				return false
			}
			return true
		}
		if node.Left.GetType() == TypColName && node.Right.GetType() == TypSQLVal {
			sqlVal, _ := node.Right.(*SQLVal)
			if sqlVal.Type != IntVal {
				return false
			}
			return true
		}
	}
	return false
}

func (node *ComparisonExpr) IsIN() bool {
	return node.Operator == InStr || node.Operator == NotInStr
}

func (node *ComparisonExpr) IsLike() bool {
	return node.Operator == LikeStr || node.Operator == NotLikeStr
}

func (node *ComparisonExpr) IsRegexp() bool {
	return node.Operator == RegexpStr || node.Operator == NotRegexpStr
}

// RownumPos return the rownum position
func (node *ComparisonExpr) RownumPos() (RNLeftOrRight, error) {
	if node == nil {
		return RNUnknown, errors.New(errors.ErrInvalidArgument, "")
	}
	lexist, rexist := false, false
	Walk(func(node SQLNode) (kontinue bool, err error) {
		switch node.(type) {
		case *RownumExpr:
			lexist = true
			return false, errors.New(errors.ErrIgnored, "exit walk")
		}
		return true, nil
	}, node.Left)
	Walk(func(node SQLNode) (kontinue bool, err error) {
		switch node.(type) {
		case *RownumExpr:
			rexist = true
			return false, errors.New(errors.ErrIgnored, "exit walk")
		}
		return true, nil
	}, node.Right)
	if lexist && rexist {
		return RNinBoth, nil
	} else if !lexist && rexist {
		return RNinRight, nil
	} else if lexist && !rexist {
		return RNinLeft, nil
	}
	return RNUnknown, nil
}

// Format formats the node.
func (node *ComparisonExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v %s %v", node.Left, node.Operator, node.Right)
	if node.Escape != nil {
		buf.Myprintf(" escape %v", node.Escape)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *ComparisonExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Left,
		node.Right,
		node.Escape,
	)
}

// ConverToLimit return a constructed Limit operator according to expression
// magic
// see: http://172.16.1.168:8090/pages/viewpage.action?pageId=24592854
func (node *ComparisonExpr) ConvertToLimit() (*Limit, error) {
	emptySet := NewLimit(0, 0)
	pos, err := node.RownumPos()
	if err != nil {
		return nil, err
	}
	if pos == RNinBoth || pos == RNUnknown {
		return nil, errors.New(errors.ErrNotSupport, "not support")
	}
	// this check avoid arithmetic or rownum
	if pos == RNinLeft {
		if node.Left.GetType() != TypRownum {
			return nil, errors.New(errors.ErrNotSupport, "not support")
		}
	} else if pos == RNinRight {
		if node.Right.GetType() != TypRownum {
			return nil, errors.New(errors.ErrNotSupport, "not support")
		}
	}

	if node.IsLike() || node.IsRegexp() || node.Operator == JSONExtractOp || node.Operator == JSONUnquoteExtractOp || node.Operator == NullSafeEqualStr {
		return nil, errors.New(errors.ErrNotSupport, "not support")
	}
	if node.IsIN() {
		if pos != RNinLeft {
			return nil, errors.New(errors.ErrNotSupport, fmt.Sprintf("what can be in rownum, node is %s", node.String()))
		}
		valTuple, ok := node.Right.(ValTuple)
		if !ok {
			return nil, errors.New(errors.ErrNotSupport, "not support")
		}
		// we need parse int from listArg
		rnSet, err := valTuple.ParseInt()
		if err != nil {
			return nil, err
		}
		if len(rnSet) == 0 {
			return nil, fmt.Errorf("no element follows 'in'")
		}
		min, max := SearchMin(rnSet)
		if node.Operator == InStr {
			if min != 1 {
				return emptySet, nil
			}
			return NewLimit(0, max), nil
		} else {
			if min == 1 {
				return emptySet, nil
			} else if min == 0 {
				return nil, nil
			}
			return NewLimit(0, min-1), nil
		}
	}
	// target is int(if pos is left, then target is right )
	target := node.Left
	if pos == RNinLeft {
		target = node.Right
	}
	sqlVal, ok := target.(*SQLVal)
	if !ok {
		return nil, errors.New(errors.ErrNotSupport, "not support")
	}
	if sqlVal.Type != IntVal {
		return nil, errors.New(errors.ErrNotSupport, "not support")
	}
	x, err := strconv.Atoi(string(sqlVal.Val))
	if err != nil {
		return nil, err
	}
	switch node.Operator {
	case EqualStr:
		if x != 1 {
			return emptySet, nil
		}
		return NewLimit(0, 1), nil
	case LessThanStr:
		if x <= 1 {
			return emptySet, nil
		}
		return NewLimit(0, x-1), nil
	case LessEqualStr:
		if x <= 0 {
			return emptySet, nil
		}
		return NewLimit(0, x), nil
	case GreaterThanStr:
		if x < 1 {
			return nil, nil
		}
		return emptySet, nil
	case GreaterEqualStr:
		if x <= 1 {
			return nil, nil
		}
		return emptySet, nil
	case NotEqualStr:
		if x < 1 {
			return nil, nil
		}
		return NewLimit(0, x-1), nil
	}
	return nil, errors.New(errors.ErrNotSupport, "not support")
}

// Compare return true of false
func (node *ComparisonExpr) Compare(v int) (bool, error) {
	var foo int
	if sqlVal, ok := node.Left.(*SQLVal); ok {
		if sqlVal.Type == IntVal {
			intVal, err := strconv.Atoi(string(sqlVal.Val))
			if err != nil {
				return false, err
			}
			foo = intVal
		} else {
			return false, errors.New(errors.ErrNotSupport, "only support int type")
		}
	} else if sqlVal, ok := node.Right.(*SQLVal); ok {
		if sqlVal.Type == IntVal {
			intVal, err := strconv.Atoi(string(sqlVal.Val))
			if err != nil {
				return false, err
			}
			foo = intVal
		} else {
			return false, errors.New(errors.ErrNotSupport, "only support int type")
		}
	} else {
		return false, fmt.Errorf("expr[%+v] is not support", node)
	}
	switch node.Operator {
	case EqualStr:
		return v == foo, nil
	case LessThanStr:
		return v < foo, nil
	case LessEqualStr:
		return v <= foo, nil
	case GreaterThanStr:
		return v > foo, nil
	case GreaterEqualStr:
		return v >= foo, nil
	case NotEqualStr:
		return v != foo, nil
	default:
		return false, fmt.Errorf("operator[%s] is not support", node.Operator)
	}
}

// RangeCond represents a BETWEEN or a NOT BETWEEN expression.
type RangeCond struct {
	Operator string
	Left     Expr
	From, To Expr
}

// RangeCond.Operator
const (
	BetweenStr    = "between"
	NotBetweenStr = "not between"
)

// Format formats the node.
func (node *RangeCond) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v %s %v and %v", node.Left, node.Operator, node.From, node.To)
}

// WalkSubtree walks the nodes of the subtree.
func (node *RangeCond) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Left,
		node.From,
		node.To,
	)
}

// IsExpr represents an IS ... or an IS NOT ... expression.
type IsExpr struct {
	Operator string
	Expr     Expr
}

// IsExpr.Operator
const (
	IsNullStr     = "is null"
	IsNotNullStr  = "is not null"
	IsTrueStr     = "is true"
	IsNotTrueStr  = "is not true"
	IsFalseStr    = "is false"
	IsNotFalseStr = "is not false"
)

// Format formats the node.
func (node *IsExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v %s", node.Expr, node.Operator)
}

// WalkSubtree walks the nodes of the subtree.
func (node *IsExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Expr,
	)
}

// ExistsExpr represents an EXISTS expression.
type ExistsExpr struct {
	Subquery *Subquery
}

// Format formats the node.
func (node *ExistsExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("exists %v", node.Subquery)
}

// WalkSubtree walks the nodes of the subtree.
func (node *ExistsExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Subquery,
	)
}

// ValType specifies the type for SQLVal.
type ValType int

// These are the possible Valtype values.
// HexNum represents a 0x... value. It cannot
// be treated as a simple value because it can
// be interpreted differently depending on the
// context.
const (
	StrVal = ValType(iota)
	IntVal
	FloatVal
	HexNum
	HexVal
	ValArg
	BitVal
)

type DefaultNullVal interface {
	iDefaultNull()
}

// SQLVal represents a single value.
type SQLVal struct {
	Type ValType
	Val  []byte
}

// NewStrVal builds a new StrVal.
func NewStrVal(in []byte) *SQLVal {
	return &SQLVal{Type: StrVal, Val: in}
}

// NewIntVal builds a new IntVal.
func NewIntVal(in []byte) *SQLVal {
	return &SQLVal{Type: IntVal, Val: in}
}

// NewFloatVal builds a new FloatVal.
func NewFloatVal(in []byte) *SQLVal {
	return &SQLVal{Type: FloatVal, Val: in}
}

// NewHexNum builds a new HexNum.
func NewHexNum(in []byte) *SQLVal {
	return &SQLVal{Type: HexNum, Val: in}
}

// NewHexVal builds a new HexVal.
func NewHexVal(in []byte) *SQLVal {
	return &SQLVal{Type: HexVal, Val: in}
}

// NewBitVal builds a new BitVal containing a bit literal.
func NewBitVal(in []byte) *SQLVal {
	return &SQLVal{Type: BitVal, Val: in}
}

// NewValArg builds a new ValArg.
func NewValArg(in []byte) *SQLVal {
	return &SQLVal{Type: ValArg, Val: in}
}

// NewSQLBoolVal builds a new SQLBoolVal
func NewSQLBoolVal(in BoolVal) *SQLVal {
	if in {
		return &SQLVal{Type: IntVal, Val: []byte("1")}
	}
	return &SQLVal{Type: IntVal, Val: []byte("0")}
}

// Format formats the node.
func (node *SQLVal) Format(buf *TrackedBuffer) {
	switch node.Type {
	case StrVal:
		sqltypes.MakeTrusted(sqltypes.VarBinary, node.Val).EncodeSQL(buf)
	case IntVal, FloatVal, HexNum:
		buf.Myprintf("%s", node.Val)
	case HexVal:
		buf.Myprintf("X'%s'", node.Val)
	case BitVal:
		buf.Myprintf("B'%s'", node.Val)
	case ValArg:
		buf.WriteArg(string(node.Val))
	default:
		panic("unexpected")
	}
}

func (node *SQLVal) iDefaultNull() {}

// WalkSubtree walks the nodes of the subtree.
func (node *SQLVal) WalkSubtree(visit Visit) error {
	return nil
}

// HexDecode decodes the hexval into bytes.
func (node *SQLVal) HexDecode() ([]byte, error) {
	dst := make([]byte, hex.DecodedLen(len(node.Val)))
	_, err := hex.Decode(dst, node.Val)
	if err != nil {
		return nil, err
	}
	return dst, err
}

// NullVal represents a NULL value.
type NullVal struct{}

// Format formats the node.
func (node *NullVal) Format(buf *TrackedBuffer) {
	buf.Myprintf("null")
}

// WalkSubtree walks the nodes of the subtree.
func (node *NullVal) WalkSubtree(visit Visit) error {
	return nil
}

// BoolVal is true or false.
type BoolVal bool

// Format formats the node.
func (node BoolVal) Format(buf *TrackedBuffer) {
	if node {
		buf.Myprintf("true")
	} else {
		buf.Myprintf("false")
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node BoolVal) WalkSubtree(visit Visit) error {
	return nil
}

func (node BoolVal) iDefaultNull() {}

// ColName represents a column name.
type ColName struct {
	// Metadata is not populated by the parser.
	// It's a placeholder for analyzers to store
	// additional data, typically info about which
	// table or column this node references.
	Metadata  interface{}
	Name      ColIdent
	Qualifier TableName
}

// Format formats the node.
func (node *ColName) Format(buf *TrackedBuffer) {
	if !node.Qualifier.IsEmpty() {
		buf.Myprintf("%v.", node.Qualifier)
	}
	buf.Myprintf("%v", node.Name)
}

// WalkSubtree walks the nodes of the subtree.
func (node *ColName) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Name,
		node.Qualifier,
	)
}

// Equal returns true if the column names match.
func (node *ColName) Equal(c *ColName) bool {
	// Failsafe: ColName should not be empty.
	if node == nil || c == nil {
		return false
	}
	return node.Name.Equal(c.Name) && node.Qualifier == c.Qualifier
}

// ColTuple represents a list of column values.
// It can be ValTuple, Subquery, ListArg.
type ColTuple interface {
	iColTuple()
	Expr
}

func (ValTuple) iColTuple()  {}
func (*Subquery) iColTuple() {}
func (ListArg) iColTuple()   {}

// ValTuple represents a tuple of actual values.
type ValTuple Exprs

// Format formats the node.
func (node ValTuple) Format(buf *TrackedBuffer) {
	buf.Myprintf("(%v)", Exprs(node))
}

// WalkSubtree walks the nodes of the subtree.
func (node ValTuple) WalkSubtree(visit Visit) error {
	return Walk(visit, Exprs(node))
}

func (node ValTuple) ParseInt() ([]int, error) {
	var ans []int
	for _, expr := range node {
		temp, err := ParseInt(expr)
		if err != nil {
			return nil, err
		}
		ans = append(ans, temp)
	}
	return ans, nil
}

// Subquery represents a subquery.
type Subquery struct {
	Select SelectStatement
}

// Format formats the node.
func (node *Subquery) Format(buf *TrackedBuffer) {
	buf.Myprintf("(%v)", node.Select)
}

// WalkSubtree walks the nodes of the subtree.
func (node *Subquery) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Select,
	)
}

// ListArg represents a named list argument.
type ListArg []byte

// Format formats the node.
func (node ListArg) Format(buf *TrackedBuffer) {
	buf.WriteArg(string(node))
}

// WalkSubtree walks the nodes of the subtree.
func (node ListArg) WalkSubtree(visit Visit) error {
	return nil
}

// BinaryExpr represents a binary value expression.
type BinaryExpr struct {
	Operator    string
	Left, Right Expr
}

// BinaryExpr.Operator
const (
	BitAndStr     = "&"
	BitOrStr      = "|"
	BitXorStr     = "^"
	PlusStr       = "+"
	MinusStr      = "-"
	MultStr       = "*"
	DivStr        = "/"
	IntDivStr     = "div"
	ModStr        = "%"
	ShiftLeftStr  = "<<"
	ShiftRightStr = ">>"
)

// Format formats the node.
func (node *BinaryExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v %s %v", node.Left, node.Operator, node.Right)
}

// WalkSubtree walks the nodes of the subtree.
func (node *BinaryExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Left,
		node.Right,
	)
}

// UnaryExpr represents a unary value expression.
type UnaryExpr struct {
	Operator string
	Expr     Expr
}

// UnaryExpr.Operator
const (
	UPlusStr  = "+"
	UMinusStr = "-"
	TildaStr  = "~"
	BangStr   = "!"
	BinaryStr = "binary "
)

// Format formats the node.
func (node *UnaryExpr) Format(buf *TrackedBuffer) {
	if _, unary := node.Expr.(*UnaryExpr); unary {
		buf.Myprintf("%s %v", node.Operator, node.Expr)
		return
	}
	buf.Myprintf("%s%v", node.Operator, node.Expr)
}

// WalkSubtree walks the nodes of the subtree.
func (node *UnaryExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Expr,
	)
}

// IntervalExpr represents a date-time INTERVAL expression.
type IntervalExpr struct {
	Expr Expr
	Unit ColIdent
}

// Format formats the node.
func (node *IntervalExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("interval %v %v", node.Expr, node.Unit)
}

// WalkSubtree walks the nodes of the subtree.
func (node *IntervalExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Expr,
		node.Unit,
	)
}

// CollateExpr represents dynamic collate operator.
type CollateExpr struct {
	Expr    Expr
	Charset string
}

// Format formats the node.
func (node *CollateExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v collate %s", node.Expr, node.Charset)
}

// WalkSubtree walks the nodes of the subtree.
func (node *CollateExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Expr,
	)
}

func CheckFunc(expr Expr) error {
	if e, ok := expr.(*FuncExpr); ok {
		if e.IsOracleFunc() {
			return e.checkSelf()
		}
	}
	return nil
}

// FuncExpr represents a function call.
type FuncExpr struct {
	Qualifier     TableIdent
	Name          ColIdent
	Distinct      bool
	Exprs         SelectExprs
	formatOriName bool
}

func (node *FuncExpr) checkSelf() error {
	count := len(node.Exprs)
	switch node.Name.Lowered() {
	case "nvl":
		if count != 2 {
			return fmt.Errorf(InvalidNumOfArguments)
		}
	case "nvl2":
		if count != 3 {
			return fmt.Errorf(InvalidNumOfArguments)
		}
	}
	return nil
}

// OracleFunc is a map of all oracle function
var OracleFunc = map[string]bool{
	"nvl":  true,
	"nvl2": true,
	// more ...
}

func (node *FuncExpr) IsOracleFunc() bool {
	if node == nil {
		return false
	}
	return OracleFunc[node.Name.Lowered()]
}

func (node *FuncExpr) AddOracleAlias(buf *TrackedBuffer) {
	if buf.noAlias {
		return
	}
	switch node.Name.Lowered() {
	case "nvl":
		buf.noAlias = true
		defer func() {
			buf.noAlias = false
		}()
		switch n := node.Exprs[0].(type) {
		case *AliasedExpr:
			if e, ok := n.Expr.(*FuncExpr); ok {
				if e.IsOracleFunc() {
					e.formatOriName = true
					defer func() {
						e.formatOriName = false
					}()
				}
			}
		}
		buf.Myprintf(" AS `NVL(%v)`", node.Exprs)
	case "nvl2":
		buf.noAlias = true
		defer func() {
			buf.noAlias = false
		}()
		switch n := node.Exprs[0].(type) {
		case *AliasedExpr:
			if e, ok := n.Expr.(*FuncExpr); ok {
				if e.IsOracleFunc() {
					e.formatOriName = true
					defer func() {
						e.formatOriName = false
					}()
				}
			}
		}
		buf.Myprintf(" AS `NVL2(%v)`", node.Exprs)
	}
}

func (node *FuncExpr) rewriteOracleFunc(buf *TrackedBuffer) {
	switch node.Name.Lowered() {
	case "nvl":
		if !buf.noAlias {
			buf.noAlias = true
			defer func() {
				buf.noAlias = false
			}()
		}
		buf.Myprintf("IF((%v!='') OR (%v)=CONVERT(0,CHAR), %v, %v)", node.Exprs[0], node.Exprs[0], node.Exprs[0], node.Exprs[1])
	case "nvl2":
		if !buf.noAlias {
			buf.noAlias = true
			defer func() {
				buf.noAlias = false
			}()
		}
		buf.Myprintf("IF((%v!='') OR (%v)=CONVERT(0,CHAR), %v, %v)", node.Exprs[0], node.Exprs[0], node.Exprs[1], node.Exprs[2])
	}
}

// Format formats the node.
func (node *FuncExpr) Format(buf *TrackedBuffer) {
	var distinct string
	if node.Distinct {
		distinct = "distinct "
	}
	if !node.Qualifier.IsEmpty() {
		buf.Myprintf("%v.", node.Qualifier)
	}
	if node.IsOracleFunc() && !node.formatOriName {
		node.rewriteOracleFunc(buf)
		return
	}
	// Function names should not be back-quoted even
	// if they match a reserved word. So, print the
	// name as is.
	buf.Myprintf("%s(%s%v)", node.Name.String(), distinct, node.Exprs)
}

// WalkSubtree walks the nodes of the subtree.
func (node *FuncExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Qualifier,
		node.Name,
		node.Exprs,
	)
}

// Aggregates is a map of all aggregate functions.
var Aggregates = map[string]bool{
	"avg":          true,
	"bit_and":      true,
	"bit_or":       true,
	"bit_xor":      true,
	"count":        true,
	"group_concat": true,
	"max":          true,
	"min":          true,
	"std":          true,
	"stddev_pop":   true,
	"stddev_samp":  true,
	"stddev":       true,
	"sum":          true,
	"var_pop":      true,
	"var_samp":     true,
	"variance":     true,
}

// IsAggregate returns true if the function is an aggregate.
func (node *FuncExpr) IsAggregate() bool {
	return Aggregates[node.Name.Lowered()]
}

// GroupConcatExpr represents a call to GROUP_CONCAT
type GroupConcatExpr struct {
	Distinct  string
	Exprs     SelectExprs
	OrderBy   OrderBy
	Separator string
}

// Format formats the node
func (node *GroupConcatExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("group_concat(%s%v%v%s)", node.Distinct, node.Exprs, node.OrderBy, node.Separator)
}

// WalkSubtree walks the nodes of the subtree.
func (node *GroupConcatExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Exprs,
		node.OrderBy,
	)
}

// ConcatExpr represents a call to CONCAT
type ConcatExpr struct {
	Left, Right Expr
	Exprs       Exprs
}

// Format formats the node
func (node *ConcatExpr) Format(buf *TrackedBuffer) {
	if len(node.Exprs) == 0 {
		buf.Myprintf("concat(%v,%v)", node.Left, node.Right)
	} else {
		buf.Myprintf("concat(")
		first := true
		for _, expr := range node.Exprs {
			if first {
				first = false
			} else {
				buf.Myprintf(",")
			}
			buf.Myprintf("%v", expr)
		}
		buf.Myprintf(")")
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *ConcatExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if len(node.Exprs) == 0 {
		return Walk(
			visit,
			node.Left,
			node.Right,
		)
	}
	return Walk(visit, node.Exprs)
}

// ValuesFuncExpr represents a function call.
type ValuesFuncExpr struct {
	Name ColIdent
}

// Format formats the node.
func (node *ValuesFuncExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("values(%s)", node.Name.String())
}

// WalkSubtree walks the nodes of the subtree.
func (node *ValuesFuncExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Name,
	)
}

// ConvertExpr represents a call to CONVERT(expr, type)
// or it's equivalent CAST(expr AS type). Both are rewritten to the former.
type ConvertExpr struct {
	Expr Expr
	Type *ConvertType
}

// Format formats the node.
func (node *ConvertExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("convert(%v, %v)", node.Expr, node.Type)
}

// WalkSubtree walks the nodes of the subtree.
func (node *ConvertExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Expr,
		node.Type,
	)
}

// ConvertUsingExpr represents a call to CONVERT(expr USING charset).
type ConvertUsingExpr struct {
	Expr Expr
	Type string
}

// Format formats the node.
func (node *ConvertUsingExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("convert(%v using %s)", node.Expr, node.Type)
}

// WalkSubtree walks the nodes of the subtree.
func (node *ConvertUsingExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Expr,
	)
}

// ConvertType represents the type in call to CONVERT(expr, type)
type ConvertType struct {
	Type     string
	Length   *SQLVal
	Scale    *SQLVal
	Operator string
	Charset  string
}

// this string is "character set" and this comment is required
const (
	CharacterSetStr = " character set"
)

// Format formats the node.
func (node *ConvertType) Format(buf *TrackedBuffer) {
	buf.Myprintf("%s", node.Type)
	if node.Length != nil {
		buf.Myprintf("(%v", node.Length)
		if node.Scale != nil {
			buf.Myprintf(", %v", node.Scale)
		}
		buf.Myprintf(")")
	}
	if node.Charset != "" {
		buf.Myprintf("%s %s", node.Operator, node.Charset)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *ConvertType) WalkSubtree(visit Visit) error {
	return nil
}

// MatchExpr represents a call to the MATCH function
type MatchExpr struct {
	Columns SelectExprs
	Expr    Expr
	Option  string
}

// MatchExpr.Option
const (
	BooleanModeStr                           = " in boolean mode"
	NaturalLanguageModeStr                   = " in natural language mode"
	NaturalLanguageModeWithQueryExpansionStr = " in natural language mode with query expansion"
	QueryExpansionStr                        = " with query expansion"
)

// Format formats the node
func (node *MatchExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("match(%v) against (%v%s)", node.Columns, node.Expr, node.Option)
}

// WalkSubtree walks the nodes of the subtree.
func (node *MatchExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Columns,
		node.Expr,
	)
}

// CaseExpr represents a CASE expression.
type CaseExpr struct {
	Expr  Expr
	Whens []*When
	Else  Expr
}

// Format formats the node.
func (node *CaseExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("case ")
	if node.Expr != nil {
		buf.Myprintf("%v ", node.Expr)
	}
	for _, when := range node.Whens {
		buf.Myprintf("%v ", when)
	}
	if node.Else != nil {
		buf.Myprintf("else %v ", node.Else)
	}
	buf.Myprintf("end")
}

// WalkSubtree walks the nodes of the subtree.
func (node *CaseExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	if err := Walk(visit, node.Expr); err != nil {
		return err
	}
	for _, n := range node.Whens {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}
	if err := Walk(visit, node.Else); err != nil {
		return err
	}
	return nil
}

// Default represents a DEFAULT expression.
type Default struct {
	ColName string
}

// Format formats the node.
func (node *Default) Format(buf *TrackedBuffer) {
	buf.Myprintf("default")
	if node.ColName != "" {
		buf.Myprintf("(%s)", node.ColName)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *Default) WalkSubtree(visit Visit) error {
	return nil
}

// When represents a WHEN sub-expression.
type When struct {
	Cond Expr
	Val  Expr
}

// Format formats the node.
func (node *When) Format(buf *TrackedBuffer) {
	buf.Myprintf("when %v then %v", node.Cond, node.Val)
}

// WalkSubtree walks the nodes of the subtree.
func (node *When) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Cond,
		node.Val,
	)
}

// GroupBy represents a GROUP BY clause.
type GroupBy []Expr

// Format formats the node.
func (node GroupBy) Format(buf *TrackedBuffer) {
	prefix := " group by "
	for _, n := range node {
		buf.Myprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node GroupBy) WalkSubtree(visit Visit) error {
	for _, n := range node {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}
	return nil
}

// OrderBy represents an ORDER By clause.
type OrderBy []*Order

// Format formats the node.
func (node OrderBy) Format(buf *TrackedBuffer) {
	prefix := " order by "
	for _, n := range node {
		buf.Myprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node OrderBy) WalkSubtree(visit Visit) error {
	for _, n := range node {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}
	return nil
}

// Order represents an ordering expression.
type Order struct {
	Expr      Expr
	Direction string
}

// Order.Direction
const (
	AscScr  = "asc"
	DescScr = "desc"
)

// Format formats the node.
func (node *Order) Format(buf *TrackedBuffer) {
	if node, ok := node.Expr.(*NullVal); ok {
		buf.Myprintf("%v", node)
		return
	}
	buf.Myprintf("%v %s", node.Expr, node.Direction)
}

// WalkSubtree walks the nodes of the subtree.
func (node *Order) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Expr,
	)
}

// Limit represents a LIMIT clause.
type Limit struct {
	Offset, Rowcount Expr
}

// Format formats the node.
func (node *Limit) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	buf.Myprintf(" limit ")
	if node.Offset != nil {
		buf.Myprintf("%v, ", node.Offset)
	}
	buf.Myprintf("%v", node.Rowcount)
}

// WalkSubtree walks the nodes of the subtree.
func (node *Limit) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Offset,
		node.Rowcount,
	)
}

// NewLimit new a limit
func NewLimit(offset, rowcount int) *Limit {
	return &Limit{
		Offset:   NewIntVal([]byte(fmt.Sprintf("%d", offset))),
		Rowcount: NewIntVal([]byte(fmt.Sprintf("%d", rowcount))),
	}
}

// MergeLimits merge many to one
func MergeLimits(limits []*Limit) (*Limit, error) {
	var min, max []int
	if len(limits) == 0 {
		return nil, nil
	}
	for _, limit := range limits {
		if limit == nil {
			continue
		}
		offset, err := ParseInt(limit.Offset)
		if err != nil {
			return nil, err
		}
		rowcount, err := ParseInt(limit.Rowcount)
		if err != nil {
			return nil, err
		}
		min = append(min, offset)
		max = append(max, offset+rowcount)
	}
	sort.Ints(min)
	sort.Ints(max)
	if len(min) < 1 || len(min) != len(max) {
		return nil, fmt.Errorf("unexpected, len(min)=%d, len(max)=%d", len(min), len(max))
	}
	minMax := min[len(min)-1]
	maxMin := max[0]
	if minMax > maxMin {
		return NewLimit(0, 0), nil
	}
	return NewLimit(minMax, maxMin-minMax), nil
}

// ParseLimit parse a range
func ParseLimit(l *Limit) (int, int, error) {
	if l == nil {
		return 0, math.MaxInt64, nil
	}
	min, err := ParseInt(l.Offset)
	if err != nil {
		return 0, 0, err
	}
	count, err := ParseInt(l.Rowcount)
	if err != nil {
		return 0, 0, err
	}
	return min, min + count, nil
}

// Values represents a VALUES clause.
type Values []ValTuple

// GetColumnVal returns value of column in insert/update
func (node Values) GetColumnVal(index int) ([]int, error) {
	var res []int
	if node == nil {
		return nil, errors.New(errors.ErrInvalidArgument, "invalid parameters")
	}
	if index < 0 {
		return nil, errors.New(errors.ErrInvalidArgument, "index should be positive integer")
	}
	for _, tuple := range node {
		if len(tuple) < index+1 {
			return nil, errors.New(errors.ErrInvalidArgument, fmt.Sprintf("target index is %d, while length is %d", index, len(tuple)))
		}
		expr := tuple[index]
		switch e := expr.(type) {
		case *SQLVal:
			if e.Type == IntVal {
				intVal, err := strconv.Atoi(string(e.Val))
				if err != nil {
					return nil, err
				}
				res = append(res, intVal)
			} else {
				return nil, errors.New(errors.ErrNotSupport, "only support int type")
			}
		default:
			return nil, errors.New(errors.ErrNotSupport, "not support")
		}
	}
	return res, nil
}

func (node Values) GetBindVarKey(index int) ([]string, error) {
	var res []string
	if node == nil {
		return nil, errors.New(errors.ErrInvalidArgument, "invalid parameters")
	}
	if index < 0 {
		return nil, errors.New(errors.ErrInvalidArgument, "index should be positive integer")
	}
	for _, tuple := range node {
		if len(tuple) < index+1 {
			return nil, errors.New(errors.ErrInvalidArgument, fmt.Sprintf("target index is %d, while length is %d", index, len(node)))
		}
		expr := tuple[index]
		switch se := expr.(type) {
		case *SQLVal:
			if se.Type == ValArg {
				res = append(res, string(se.Val))
			} else {
				return nil, errors.New(errors.ErrNotSupport, "only support int type")
			}
		default:
			return nil, errors.New(errors.ErrNotSupport, "not support")
		}
	}
	return res, nil
}

// Format formats the node.
func (node Values) Format(buf *TrackedBuffer) {
	prefix := "values "
	for _, n := range node {
		buf.Myprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node Values) WalkSubtree(visit Visit) error {
	for _, n := range node {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}
	return nil
}

// UpdateExprs represents a list of update expressions.
type UpdateExprs []*UpdateExpr

// Format formats the node.
func (node UpdateExprs) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Myprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node UpdateExprs) WalkSubtree(visit Visit) error {
	for _, n := range node {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}
	return nil
}

// UpdateExpr represents an update expression.
type UpdateExpr struct {
	Name *ColName
	Expr Expr
}

// Format formats the node.
func (node *UpdateExpr) Format(buf *TrackedBuffer) {
	buf.Myprintf("%v = %v", node.Name, node.Expr)
}

// WalkSubtree walks the nodes of the subtree.
func (node *UpdateExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Name,
		node.Expr,
	)
}

// SetExprs represents a list of set expressions.
type SetExprs []*SetExpr

// Format formats the node.
func (node SetExprs) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Myprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node SetExprs) WalkSubtree(visit Visit) error {
	for _, n := range node {
		if err := Walk(visit, n); err != nil {
			return err
		}
	}
	return nil
}

// SetExpr represents a set expression.
type SetExpr struct {
	Name ColIdent
	Expr Expr
}

// SetExpr.Expr, for SET TRANSACTION ... or START TRANSACTION
const (
	// TransactionStr is the Name for a SET TRANSACTION statement
	TransactionIsolationLevelStr = "transaction isolation level"
	TransactionStr               = "transaction"
	TxStr                        = "tx_isolation"

	IsolationLevelReadUncommitted = "read uncommitted"
	IsolationLevelReadCommitted   = "read committed"
	IsolationLevelRepeatableRead  = "repeatable read"
	IsolationLevelSerializable    = "serializable"

	TxReadOnly  = "read only"
	TxReadWrite = "read write"
)

// Format formats the node.
func (node *SetExpr) Format(buf *TrackedBuffer) {
	// We don't have to backtick set variable names.
	if node.Name.EqualString("charset") || node.Name.EqualString("names") {
		buf.Myprintf("%s %v", node.Name.String(), node.Expr)
	} else if node.Name.EqualString(TransactionIsolationLevelStr) || node.Name.EqualString(TransactionStr) {
		sqlVal := node.Expr.(*SQLVal)
		buf.Myprintf("%s %s", node.Name.String(), strings.ToLower(string(sqlVal.Val)))
	} else {
		buf.Myprintf("%s = %v", node.Name.String(), node.Expr)
	}
}

// WalkSubtree walks the nodes of the subtree.
func (node *SetExpr) WalkSubtree(visit Visit) error {
	if node == nil {
		return nil
	}
	return Walk(
		visit,
		node.Name,
		node.Expr,
	)
}

// OnDup represents an ON DUPLICATE KEY clause.
type OnDup UpdateExprs

// Format formats the node.
func (node OnDup) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	buf.Myprintf(" on duplicate key update %v", UpdateExprs(node))
}

// WalkSubtree walks the nodes of the subtree.
func (node OnDup) WalkSubtree(visit Visit) error {
	return Walk(visit, UpdateExprs(node))
}

// ColIdent is a case insensitive SQL identifier. It will be escaped with
// backquotes if necessary.
type ColIdent struct {
	// This artifact prevents this struct from being compared
	// with itself. It consumes no space as long as it's not the
	// last field in the struct.
	_            [0]struct{ _ []byte }
	val, lowered string
}

// NewColIdent makes a new ColIdent.
func NewColIdent(str string) ColIdent {
	return ColIdent{
		val: str,
	}
}

// Format formats the node.
func (node ColIdent) Format(buf *TrackedBuffer) {
	FormatID(buf, node.val, node.Lowered(), false)
}

// WalkSubtree walks the nodes of the subtree.
func (node ColIdent) WalkSubtree(visit Visit) error {
	return nil
}

// IsEmpty returns true if the name is empty.
func (node ColIdent) IsEmpty() bool {
	return node.val == ""
}

// String returns the unescaped column name. It must
// not be used for SQL generation. Use sqlparser.String
// instead. The Stringer conformance is for usage
// in templates.
func (node ColIdent) String() string {
	return node.val
}

// CompliantName returns a compliant id name
// that can be used for a bind var.
func (node ColIdent) CompliantName() string {
	return compliantName(node.val)
}

// Lowered returns a lower-cased column name.
// This function should generally be used only for optimizing
// comparisons.
func (node ColIdent) Lowered() string {
	if node.val == "" {
		return ""
	}
	if node.lowered == "" {
		node.lowered = strings.ToLower(node.val)
	}
	return node.lowered
}

// Equal performs a case-insensitive compare.
func (node ColIdent) Equal(in ColIdent) bool {
	return node.Lowered() == in.Lowered()
}

// EqualString performs a case-insensitive compare with str.
func (node ColIdent) EqualString(str string) bool {
	return node.Lowered() == strings.ToLower(str)
}

// MarshalJSON marshals into JSON.
func (node ColIdent) MarshalJSON() ([]byte, error) {
	return json.Marshal(node.val)
}

// UnmarshalJSON unmarshals from JSON.
func (node *ColIdent) UnmarshalJSON(b []byte) error {
	var result string
	err := json.Unmarshal(b, &result)
	if err != nil {
		return err
	}
	node.val = result
	return nil
}

// TableIdent is a case sensitive SQL identifier. It will be escaped with
// backquotes if necessary.
type TableIdent struct {
	v string
}

// NewTableIdent creates a new TableIdent.
func NewTableIdent(str string) TableIdent {
	if *lowerCaseTableNames {
		str = strings.ToLower(str)
	}
	return TableIdent{v: str}
}

// Format formats the node.
func (node TableIdent) Format(buf *TrackedBuffer) {
	FormatID(buf, node.v, strings.ToLower(node.v), false)
}

// WalkSubtree walks the nodes of the subtree.
func (node TableIdent) WalkSubtree(visit Visit) error {
	return nil
}

// IsEmpty returns true if TabIdent is empty.
func (node TableIdent) IsEmpty() bool {
	return node.v == ""
}

// String returns the unescaped table name. It must
// not be used for SQL generation. Use sqlparser.String
// instead. The Stringer conformance is for usage
// in templates.
func (node TableIdent) String() string {
	return node.v
}

// CompliantName returns a compliant id name
// that can be used for a bind var.
func (node TableIdent) CompliantName() string {
	return compliantName(node.v)
}

// MarshalJSON marshals into JSON.
func (node TableIdent) MarshalJSON() ([]byte, error) {
	return json.Marshal(node.v)
}

// UnmarshalJSON unmarshals from JSON.
func (node *TableIdent) UnmarshalJSON(b []byte) error {
	var result string
	err := json.Unmarshal(b, &result)
	if err != nil {
		return err
	}
	node.v = result
	return nil
}

// Backtick produces a backticked literal given an input string.
func Backtick(in string) string {
	var buf bytes.Buffer
	buf.WriteByte('`')
	for _, c := range in {
		buf.WriteRune(c)
		if c == '`' {
			buf.WriteByte('`')
		}
	}
	buf.WriteByte('`')
	return buf.String()
}

// FormatIDUniform formats colun nae or table nae
func FormatIDUniform(buf *TrackedBuffer, original string, mustEscape bool) {
	isDbSystemVariable := false
	lowered := strings.ToLower(original)

	if mustEscape {
		goto mustEscape
	}

	if len(original) > 1 && original[:2] == "@@" {
		isDbSystemVariable = true
	}

	for i, c := range original {
		if !isLetter(uint16(c)) && (!isDbSystemVariable || !isCarat(uint16(c))) {
			if i == 0 || !isDigit(uint16(c)) {
				goto mustEscape
			}
		}
	}
	if _, ok := keywords[lowered]; ok {
		if _, doNotEscape := escapeBlacklist[lowered]; !doNotEscape {
			goto mustEscape
		}
	}
	buf.Myprintf("%s", original)
	return

mustEscape:
	buf.WriteByte('"')
	for _, c := range original {
		if c == '"' {
			buf.WriteByte('\\')
		} else if c == '\\' {
			buf.WriteByte('\\')
		}
		buf.WriteRune(c)
	}
	buf.WriteByte('"')
}

// FormatID formats column name or table name
func FormatID(buf *TrackedBuffer, original, lowered string, mustEscape bool) {
	isDbSystemVariable := false
	if mustEscape {
		goto mustEscape
	}
	if len(original) > 1 && original[:2] == "@@" {
		isDbSystemVariable = true
	}

	for i, c := range original {
		if !isLetter(uint16(c)) && (!isDbSystemVariable || !isCarat(uint16(c))) {
			if i == 0 || !isDigit(uint16(c)) {
				goto mustEscape
			}
		}
	}
	if _, ok := keywords[lowered]; ok {
		if _, doNotEscape := escapeBlacklist[lowered]; !doNotEscape {
			goto mustEscape
		}
	}
	buf.Myprintf("%s", original)
	return

mustEscape:
	buf.WriteByte('`')
	for _, c := range original {
		buf.WriteRune(c)
		if c == '`' {
			buf.WriteByte('`')
		}
	}
	buf.WriteByte('`')
}

func compliantName(in string) string {
	var buf bytes.Buffer
	for i, c := range in {
		if !isLetter(uint16(c)) {
			if i == 0 || !isDigit(uint16(c)) {
				buf.WriteByte('_')
				continue
			}
		}
		buf.WriteRune(c)
	}
	return buf.String()
}

// NodeHasRownum returns true if the node contains RownumExpr
func NodeHasRownum(node SQLNode) bool {
	hasRownum := false
	_ = Walk(func(node SQLNode) (kontinue bool, err error) {
		switch node.(type) {
		case *RownumExpr:
			hasRownum = true
			return false, errors.New(errors.ErrIgnored, "unused error")
		case *Subquery:
			// Subqueries are analyzed by themselves.
			return false, nil
		}
		return true, nil
	}, node)
	return hasRownum
}

// NodeIsRownum returns true if the node is RownumExpr
func NodeIsRownum(node SQLNode) bool {
	if _, ok := node.(*RownumExpr); ok {
		return true
	}
	return false
}

// NodeIsIntVal returns true if the node is IntVal
func NodeIsIntVal(node SQLNode) bool {
	if sqlVal, ok := node.(*SQLVal); ok {
		if sqlVal.Type == IntVal {
			return true
		}
	}
	return false
}

// CountRownum returns the count of rownum
func CountRownum(node SQLNode) int {
	ans := 0
	_ = Walk(func(node SQLNode) (kontinue bool, err error) {
		switch node.(type) {
		case *RownumExpr:
			ans++
		}
		return true, nil
	}, node)
	return ans
}

// NodeHasAssignExpr returns true if the node contains AssignExpr
func NodeHasAssignExpr(node SQLNode) bool {
	hasAssignExpr := false
	_ = Walk(func(node SQLNode) (kontinue bool, err error) {
		switch node.(type) {
		case *AssignExpr:
			hasAssignExpr = true
			return false, errors.New(errors.ErrIgnored, "unused error")
		case *Subquery:
			// Subqueries are analyzed by themselves.
			return false, nil
		}
		return true, nil
	}, node)
	return hasAssignExpr
}

func checkDateTimePrecisionOption(selectExprs SelectExprs) error {
	if selectExprs == nil || len(selectExprs) == 0 {
		return nil
	}
	if len(selectExprs) > 1 {
		return fmt.Errorf("invalid date time precision option, only one parameter is needed")
	}
	aliasedExpr, ok := selectExprs[0].(*AliasedExpr)
	if !ok {
		return fmt.Errorf("invalid date time precision option \"%v\", not an AliasedExpr (time precision option must be an integer between 0-6 or a normalized arg value)", String(selectExprs[0]))
	}
	val, ok := aliasedExpr.Expr.(*SQLVal)
	if !ok {
		return fmt.Errorf("invalid date time precision option \"%v\", not an SQL value (time precision option must be an integer between 0-6 or a normalized arg value)", String(aliasedExpr.Expr))
	}
	if val.Type != IntVal && val.Type != ValArg {
		return fmt.Errorf("invalid date time precision value type \"%v\" (time precision option must be an integer between 0-6 or a normalized arg value)", val.Type)
	}
	if val.Type == IntVal {
		intVal, err := strconv.Atoi(string(val.Val))
		if err != nil || intVal < 0 || intVal > 6 {
			return fmt.Errorf("invalid date time precision integer value \"%v\" (time precision option must be an integer between 0-6)", intVal)
		}
	}
	return nil
}

var timeFunctions = make(map[string]bool)

func init() {
	timeFunctions["current_timestamp"] = false
	timeFunctions["utc_timestamp"] = false
	timeFunctions["utc_time"] = false
	timeFunctions["utc_date"] = false
	timeFunctions["localtime"] = false
	timeFunctions["localtimestamp"] = false
	timeFunctions["current_date"] = false
	timeFunctions["current_time"] = false
	timeFunctions["now"] = false
}

// IsTimeFunc indicates if the FuncExpr represents
// a time func.
func (node *FuncExpr) IsTimeFunc() bool {
	funcName := strings.ToLower(node.Name.String())
	_, ok := timeFunctions[funcName]
	return ok
}

// Explain represents an EXPLAIN statement
type Explain struct {
	Type      string
	Statement Statement
}

// Format formats the node.
func (node *Explain) Format(buf *TrackedBuffer) {
	format := ""
	switch node.Type {
	case "": // do nothing
	case AnalyzeStr:
		format = AnalyzeStr + " "
	default:
		format = "format = " + node.Type + " "
	}
	buf.Myprintf("explain %s%v", format, node.Statement)
}

// WalkSubtree walks the nodes of the subtree.
func (e *Explain) WalkSubtree(visit Visit) error {
	return Walk(visit, e.Statement)
}

// SplitAndExpression breaks up the Expr into AND-separated conditions
// and appends them to filters, which can be shuffled and recombined
// as needed.
// 2020/07/29 move here from builder package, we will use it in sqlparser package
func SplitAndExpression(filters []Expr, node Expr) []Expr {
	if node == nil {
		return filters
	}
	if andNode, ok := node.(*AndExpr); ok {
		filters = SplitAndExpression(filters, andNode.Left)
		return SplitAndExpression(filters, andNode.Right)
	} else if parenNode, ok := node.(*ParenExpr); ok {
		return SplitAndExpression(filters, parenNode.Expr)
	}
	return append(filters, node)
}

// RecombinedExpression recombined expression
// @todo add test case
func RecombinedExpression(filters []Expr) Expr {
	var ans Expr
	for _, filter := range filters {
		if _, ok := filter.(*OrExpr); ok {
			filter = &ParenExpr{Expr: filter}
		}
		if ans == nil {
			ans = filter
			continue
		}
		ans = &AndExpr{
			Left:  ans,
			Right: filter,
		}
	}
	return ans
}

// RemoveRownum remove rownum expr from given filters
func RemoveRownum(filters []Expr) (afterRemove []Expr, removed []Expr, err error) {
	for _, filter := range filters {
		switch filter.(type) {
		case *OrExpr:
			return filters, nil, fmt.Errorf("rownum in or-expr is not suppport")
		case *RownumExpr:
			log.Errorf("remove rownum: not support %s", filter.String())
			return filters, nil, fmt.Errorf("stand alone rownum is not support")
		case *RangeCond, *ComparisonExpr:
			if NodeHasRownum(filter) {
				removed = append(removed, filter)
			} else {
				afterRemove = append(afterRemove, filter)
			}
		default:
			afterRemove = append(afterRemove, filter)
		}
	}
	return
}

// ConvertToLimit convert the given filters to one limit
func ConvertToLimit(filters []Expr) (*Limit, error) {
	var tempLimits []*Limit
	emptySet := NewLimit(0, 0)
	for _, expr := range filters {
		switch expr := expr.(type) {
		case *ComparisonExpr:
			limit, err := expr.ConvertToLimit()
			if err != nil {
				return nil, err
			}
			tempLimits = append(tempLimits, limit)
		case *RangeCond:
			// between .. and .. and not between .. and ..
			if expr.Left.GetType() != TypRownum {
				return nil, fmt.Errorf("%s is not rownum expression", expr.String())
			}
			min, err := ParseInt(expr.From)
			if err != nil {
				return nil, err
			}
			max, err := ParseInt(expr.To)
			if err != nil {
				return nil, err
			}
			var invalid bool
			if max < min {
				invalid = true
			}
			var limit *Limit
			if expr.Operator == BetweenStr {
				if invalid || min > 1 || max < 1 {
					limit = emptySet
				} else {
					limit = NewLimit(0, max)
				}
			} else {
				if invalid {
					limit = nil
				} else {
					if min <= 1 && max > 0 {
						limit = emptySet
					} else if max <= 0 {
						limit = nil
					} else {
						limit = NewLimit(0, min-1)
					}
				}
			}
			tempLimits = append(tempLimits, limit)
		}
	}
	return MergeLimits(tempLimits)
}
