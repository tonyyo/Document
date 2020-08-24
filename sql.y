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

%{
//lint:file-ignore SA4006 Ignore all unused code, it's generated
package sqlparser

import (
  "strings"
)

func nextAliasId(yylex interface{}) string {
  yylex.(*Tokenizer).aliasID++
  return genDerivedTableAlias(yylex.(*Tokenizer).aliasID)
}

func setParseTree(yylex interface{}, stmt Statement) {
  yylex.(*Tokenizer).ParseTree = stmt
}

func setAllowComments(yylex interface{}, allow bool) {
  yylex.(*Tokenizer).AllowComments = allow
}

func setHasSequenceUpdateNode(yylex interface{}, has bool) {
  yylex.(*Tokenizer).ExtraParseInfo.HasSequenceUpdateNode = has
}

func setHasRownum(yylex interface{}, has bool) {
  yylex.(*Tokenizer).ExtraParseInfo.HasRownum = has
}

func setHasLimit(yylex interface{}, has bool) {
  yylex.(*Tokenizer).ExtraParseInfo.HasLimit = has
}

func setDDL(yylex interface{}, ddl *DDL) {
  yylex.(*Tokenizer).partialDDL = ddl
}

func incNesting(yylex interface{}) bool {
  yylex.(*Tokenizer).nesting++
  if yylex.(*Tokenizer).nesting == 200 {
    return true
  }
  return false
}

func decNesting(yylex interface{}) {
  yylex.(*Tokenizer).nesting--
}

func forceEOF(yylex interface{}) {
  yylex.(*Tokenizer).ForceEOF = true
}

%}

%union {
  empty         struct{}
  statement     Statement
  selStmt       SelectStatement
  ddl           *DDL
  ins           *Insert
  byt           byte
  bytes         []byte
  bytes2        [][]byte
  str           string
  strs          []string
  hints         Hints
  hint          Hint
  selectExprs   SelectExprs
  selectExpr    SelectExpr
  columns       Columns
  colName       *ColName
  tableExprs    TableExprs
  tableExpr     TableExpr
  tableName     TableName
  tableNames    TableNames
  indexHints    *IndexHints
  scanMode      *ScanMode
  expr          Expr
  exprs         Exprs
  boolVal       BoolVal
  colTuple      ColTuple
  values        Values
  valTuple      ValTuple
  subquery      *Subquery
  whens         []*When
  when          *When
  orderBy       OrderBy
  order         *Order
  limit         *Limit
  updateExprs   UpdateExprs
  setExprs      SetExprs
  updateExpr    *UpdateExpr
  setExpr       *SetExpr
  colIdent      ColIdent
  colIdents     []ColIdent
  columnDetail  ColumnDetail
  columnDetails []ColumnDetail
  tableIdent    TableIdent
  convertType   *ConvertType
  aliasedTableName *AliasedTableExpr
  TableSpec  *TableSpec
  columnType    ColumnType
  colKeyOpt     ColumnKeyOption
  optVal        *SQLVal
  LengthScaleOption LengthScaleOption
  columnDefinition *ColumnDefinition
  indexDefinition *IndexDefinition
  constraintDefinition *ConstraintDefinition
  constraintDetail ConstraintDetail
  constraintStateItem *ConstraintStateItem
  constraintStateItems []*ConstraintStateItem
  indexInfo     *IndexInfo
  indexColumn   *IndexColumn
  indexColumns  []*IndexColumn
  indexOption   *IndexOption
  indexOptions   []*IndexOption
  partDefs      []*PartitionDefinition
  partDef       *PartitionDefinition
  partSpec      *PartitionSpec
  changeSpec	*ChangeSpec
  modifySpec    *ModifySpec
  kunpartDefs      []*KunPartitionDefinition
  kunpartDef       *KunPartitionDefinition
  triggerTime   string
  triggerEvent  string
  dbSpec   *DatabaseSpec
  tableFromSpec      *TableFromSpec
  showCondition      ShowCondition
  optCreatePartition *OptCreatePartition
  optLike       *OptLike
  optLocate     *OptLocate
  indexSpec     *IndexSpec
  privileges         Privileges
  object_privilege   *Privilege
  grantIdent         *GrantIdent
  keyspaceIdent      ColIdent
  principals         Principals
  principalDetail    PrincipalDetail
  principal          *Principal
  grant_option       bool
  tableOptions       TableOptions
  ref                Reference
  onUpdateDelete     OnUpdateDelete
  refAction          ReferenceAction
  procOrFuncName     ProcOrFuncName
  defaultNullVal     DefaultNullVal
  defaultNullVals    []DefaultNullVal
  alterObj           AlterObject
  alterObjTyp        AlterObjTyp
  packageSpec	     *PackageSpec
  packageBody	     *PackageBody
  procedureSpec	     *ProcedureSpec
  pLDeclaration	     PLDeclaration
  pLPackageBody	     PLPackageBody
  functionSpec	     *FunctionSpec
  parameterList	     ParameterList
  parameter	     *Parameter
  typeSpec	     *TypeSpec
  dataType	     *DataType
  variableDeclaration  *VariableDeclaration
  cursorDeclaration	*CursorDeclaration
  parameterSpec		*ParameterSpec
  parameterSpecList	ParameterSpecList
  parameterSpecType	*ParameterSpecType
  exceptionDeclaration  *ExceptionDeclaration
  subTypeDeclaration	*SubTypeDeclaration
  subTypeRange		*SubTypeRange
  plsqlExpression		*PlsqlExpression
  cursorExpression	*CursorExpression
  logicalExpression	*LogicalExpression
  typeDeclaration	*TypeDeclaration
  tableTypeDef		*TableTypeDef
  tableIndex		*TableIndex
  varrayTypeDef		*VarrayTypeDef
  recordTypeDef		*RecordTypeDef
  fieldSpecList		FieldSpecList
  fieldSpec		*FieldSpec
  cursorTypeDef		*CursorTypeDef
  procedureDefinition	*ProcedureDefinition
  procedureBody		*ProcedureBody
  callSpec		*CallSpec
  cSpec			*CSpec
  block			*Block
  declareSpec		*DeclareSpec
  body			*Body
  seqOfStatement	*SeqOfStatement
  pipeRowStatement	*PipeRowStatement
  functionCallStatement	*FunctionCallStatement
  returnStatement	*ReturnStatement
  nullStatement		*NullStatement
  raiseStatement	*RaiseStatement
  ifStatement		*IfStatement
  elsePart		*ElsePart
  elseIfList		ElseIfList
  elseIf		*ElseIf
  goToStatement		*GoToStatement
  exitStatement		*ExitStatement
  continueStatement	*ContinueStatement
  assignmentStatement	*AssignmentStatement
  generalElement	*GeneralElement
  generalElementPart	*GeneralElementPart
  generalElementParList	*GeneralElementParList
  bindVariable		*BindVariable
  idExpressionList	IdExpressionList
  idExpression		*IdExpression
  exceptionHandlerList	ExceptionHandlerList
  exceptionHandler	*ExceptionHandler
  loopStatement		*LoopStatement
  loopParam		LoopParam
  recordLoopParam	*RecordLoopParam
  indexLoopParam	*IndexLoopParam
  defaultValuePart	*DefaultValuePart
  pLStatement		PLStatement
  labelDeclaration	*LabelDeclaration
  plsqlExpressions	PlsqlExpressions
  caseStatement		*CaseStatement
  simpleCaseStatement	*SimpleCaseStatement
  searchCaseStatement	*SearchCaseStatement
  caseElsePart		*CaseElsePart
  searchCaseWhenPart	*SearchCaseWhenPart
  simpleCaseWhenPart	*SimpleCaseWhenPart
  simpleCaseWhenPartList	SimpleCaseWhenPartList
  searchCaseWhenPartList	SearchCaseWhenPartList
  executeImmediate	*ExecuteImmediate
  usingReturn		*UsingReturn
  dynamicReturn		*DynamicReturn
  intoUsing		*IntoUsing
  usingClause		*UsingClause
  usingElements		UsingElementList
  usingElement		*UsingElement
  intoClause		*IntoClause
  sqlStatement		*SqlStatement
  variableName		*VariableName
  variableNameList	VariableNameList
  cursorManuStatement	*CursorManuStatement
  openForStatement	*OpenForStatement
  fetchStatement	*FetchStatement
  openStatement		*OpenStatement
  transCtrlStatement	*TransCtrlStatement
  setTransCommand	*SetTransCommand
  setConstCommand	*SetConstCommand
  commitStatement	*CommitStatement
  rollbackStatement	*RollbackStatement
  savePointStatement	*SavePointStatement
  constraintName	*ConstraintName
  constraintNameList	ConstraintNameList
  identifier		*Identifier
  dmlStatement		*DmlStatement
  forAllStatement	*ForAllStatement
  boundsClause		*BoundsClause
  betweenBound		*BetweenBound
  indexName		*IndexName
  functionDefinition		*FunctionDefinition
  functionBody			*FunctionBody
  funcReturnSuffixList		FuncReturnSuffixList
  funcReturnSuffix		FuncReturnSuffix
  invokerRightsClause		*InvokerRightsClause
  parallelEnableClause		*ParallelEnableClause
  resultCacheClause		*ResultCacheClause
  deterministic			*Deterministic
  parenColumnList		*ParenColumnList
  streamingClause		*StreamingClause
  partitionByClause		*PartitionByClause
  partitionExtension		*PartitionExtension
  tableViewName			*TableViewName
  tableViewNameList		TableViewNameList
  reliesOnPart			*ReliesOnPart
  hierarchicalQueryClause	*HierarchicalQueryClause
  columnAttrs        ColumnAttrList
  columnAttr         ColumnAttr
  rowNumExpr         *RownumExpr
  functionAlterBody		*FunctionAlterBody
  procedureAlterBody		*ProcedureAlterBody
  packageAlterBody		*PackageAlterBody
  compilerParameterList		CompilerParameterList
  compilerParameter		*CompilerParameter
}

%token LEX_ERROR
%left <bytes> UNION
/*
  Resolve column attribute ambiguity -- force precedence of "UNIQUE KEY" against
  simple "UNIQUE" and "KEY" attributes:
*/
%right UNIQUE KEY
%token <bytes> SELECT INSERT UPDATE DELETE FROM WHERE GROUP HAVING ORDER BY LIMIT OFFSET FOR
%token <bytes> ALL DISTINCT AS EXISTS ASC DESC INTO DUPLICATE KEY DEFAULT SET LOCK INPLACE COPY NONE SHARED EXCLUSIVE PARSER
%token <bytes> VALUES LAST_INSERT_ID
%token <bytes> NEXT VALUE SHARE MODE ROWNUM
%token <bytes> SQL_NO_CACHE SQL_CACHE FETCH
%token <bytes> UNDERSCORECS
%left <bytes> JOIN STRAIGHT_JOIN LEFT RIGHT INNER OUTER CROSS NATURAL USE FORCE
%left <bytes> ON
%token <empty> '(' ',' ')' ';'
%token <bytes> ID HEX STRING INTEGRAL FLOAT HEXNUM VALUE_ARG LIST_ARG COMMENT COMMENT_KEYWORD BIT_LITERAL
%token <bytes> NULL TRUE FALSE OFF
%token <bytes> BACKEND INCEPTOR MFED GLKJOIN
%token <bytes> CALL
%token <bytes> GRANT REVOKE PRIVILEGES OPTION GRANTS USER ROLE ROUTINE USAGE EXECUTE ADMIN

// Precedence dictated by mysql. But the vitess grammar is simplified.
// Some of these operators don't conflict in our situation. Nevertheless,
// it's better to have these listed in the correct order. Also, we don't
// support all operators yet.
%left <bytes> ASSIGN
%left <bytes> OR
%left <bytes> OP_CONCAT
%left <bytes> AND
%right <bytes> NOT '!'
%left <bytes> BETWEEN CASE WHEN THEN ELSE END
%left <bytes> '=' '<' '>' LE GE NE NULL_SAFE_EQUAL IS LIKE REGEXP IN
%left <bytes> '|'
%left <bytes> '&'
%left <bytes> SHIFT_LEFT SHIFT_RIGHT
%left <bytes> '+' '-'
%left <bytes> '*' '/' DIV '%' MOD
%left <bytes> '^'
%right <bytes> '~' UNARY
%left <bytes> COLLATE
%right <bytes> BINARY
%right <bytes> INTERVAL
%nonassoc <bytes> '.'
%nonassoc <bytes> '@'
// There is no need to define precedence for the JSON
// operators because the syntax is restricted enough that
// they don't cause conflicts.
%token <empty> JSON_EXTRACT_OP JSON_UNQUOTE_EXTRACT_OP

// DDL Tokens
%token <bytes> CREATE ALTER DROP RENAME ANALYZE ADD CHANGE MODIFY
%token <bytes> TABLE INDEX VIEW TO IGNORE IF UNIQUE USING PRIMARY SEQUENCE FUNCTION TRIGGER PROCEDURE TEMPORARY PACKAGE BODY
%token <bytes> GENERATED ALWAYS VIRTUAL STORED
%token <bytes> ACTION CASCADE FOREIGN NO RESTRICT BTREE
%token <bytes> SHOW DESCRIBE EXPLAIN DATE ESCAPE REPAIR OPTIMIZE TRUNCATE
%token <bytes> MAXVALUE PARTITION REORGANIZE LESS THAN CACHE BEFORE FIRST AFTER EACH ROW RANGE LIST
%token <bytes> LOCAL FULLTEXT SPATIAL DISK MEMORY STORAGE FORMAT PARTITIONS EXTENDED COLUMN NO_WRITE_TO_BINLOG
%token <bytes> AVG_ROW_LENGTH CHECK CHECKSUM COMPRESSION CONNECTION DATA DIRECTORY DELAY_KEY_WRITE ENCRYPTION ENGINE ENGINES INSERT_METHOD KEY_BLOCK_SIZE ALGORITHM TYPE
%token <bytes> MAX_ROWS MIN_ROWS PACK_KEYS PASSWORD ROW_FORMAT STATS_AUTO_RECALC STATS_PERSISTENT STATS_SAMPLE_PAGES TABLESPACE
%token <bytes> BINARY_MD5 HASH NUMERIC_STATIC_MAP UNICODE_LOOSE_MD5 REVERSE_BITS
%token <bytes> CONSTRAINT REFERENCE REFERENCES REMOTE_SEQ REPLICATION
%token <bytes> LOCATE ERRORS WARNINGS PROCESSLIST
%token <bytes> SQL SECURITY MERGE TEMPTABLE UNDEFINED INVOKER DEFINER
%token <bytes> ENABLE DISABLE DEFERRABLE INITIALLY IMMEDIATE DEFERRED RELY NORELY VALIDATE NOVALIDATE

 // plsql tokens
%token <bytes> DOUBLE_POINT DECLARE INTRODUCER AUTHID CURRENT_USER ROWTYPE REF OUT NOCOPY CONSTANT CURSOR SUBTYPE EXCEPTION NUMBER ASSIGN_OP RETURN DETERMINISTIC RESULT_CACHE VARRAY ARRAY OF RECORD SPECIFICATION
%token <bytes> JAVA NAME EXTERNAL CONTINUE EXIT GOTO PIPE C_LETTER RAISE KEEP ELSIF WHILE LOOP REVERSE BULK COLLECT RETURNING CLOSE OPEN CLUSTER ANY BINDVAR UNSIGNED_INTEGER PERCENT_ISOPEN PERCENT_FOUND PERCENT_NOTFOUND PERCENT_ROWCOUNT
%token <bytes> SAVEPOINT CORRUPT_XID CORRUPT_XID_ALL BATCH WAIT NOWAIT CONSTRAINTS WORK SEGMENT FORALL SAVE EXCEPTIONS INDICES PIPELINED AGGREGATE PARALLEL_ENABLE RELIES_ON SUBPARTITION NOCYCLE CONNECT
%token <bytes> DEBUG REUSE SETTINGS COMPILE

 // Transaction Tokens
 %token <bytes> BEGIN START TRANSACTION COMMIT ROLLBACK

 // Explain tokens
 %token <bytes> TREE TRADITIONAL

// Type Tokens
%token <bytes> BIT TINYINT SMALLINT MEDIUMINT INT INTEGER BIGINT INTNUM
%token <bytes> REAL DOUBLE FLOAT_TYPE DECIMAL NUMERIC
%token <bytes> TIME TIMESTAMP DATETIME YEAR DAY MONTH SECOND
%token <bytes> CHAR VARCHAR BOOL CHARACTER VARBINARY NCHAR CASCADED COLLATION VARYING
%token <bytes> TEXT TINYTEXT MEDIUMTEXT LONGTEXT CLOB
%token <bytes> BLOB TINYBLOB MEDIUMBLOB LONGBLOB JSON ENUM
%token <bytes> GEOMETRY POINT LINESTRING POLYGON GEOMETRYCOLLECTION MULTIPOINT MULTILINESTRING MULTIPOLYGON

// Type Modifiers
%token <bytes> NULLX AUTO_INCREMENT APPROXNUM SIGNED UNSIGNED ZEROFILL

// Supported SHOW tokens
%token <bytes> DATABASES SCHEMAS TABLES KUNDB_KEYSPACES KUNDB_SHARDS SHARDS VSCHEMA_TABLES PACKAGES
%token <bytes> FULL COLUMNS FIELDS STATUS INDEXES KEYS TRIGGERS KUNDB_RANGE_INFO KUNDB_CHECKS KUNDB_VINDEXES VARIABLES

// SET tokens
%token <bytes> NAMES CHARSET GLOBAL SESSION ISOLATION LEVEL READ WRITE ONLY REPEATABLE COMMITTED UNCOMMITTED SERIALIZABLE

// Functions
%token <bytes> CURRENT_TIMESTAMP SCHEMA DATABASE CURRENT_DATE SERVER SYNONYM
%token <bytes> CURRENT_TIME LOCALTIME LOCALTIMESTAMP
%token <bytes> UTC_DATE UTC_TIME UTC_TIMESTAMP
%token <bytes> REPLACE
%token <bytes> CONVERT CAST
%token <bytes> GROUP_CONCAT SEPARATOR
%token <bytes> CONCAT

// Match
%token <bytes> MATCH AGAINST BOOLEAN LANGUAGE WITH QUERY EXPANSION

// MySQL reserved words that are unused by this grammar will map to this token.
%token <bytes> UNUSED

%type <statement> command
%type <selStmt> select_statement base_select union_lhs union_rhs opt_is_select
%type <statement> insert_statement update_statement delete_statement set_statement
%type <statement> create_statement alter_statement rename_statement drop_statement truncate_statement
%type <ddl> create_table_prefix
%type <ddl> create_sequence_statement
%type <statement> show_statement use_statement other_statement
%type <statement> begin_statement commit_statement rollback_statement
%type <statement> call_statement
%type <statement> dcl_statement grant_statement revoke_statement drop_user_statement create_user_statement drop_role_statement create_role_statement
%type <bytes2> opt_comment comment_list
%type <str> union_op insert_or_replace opt_with_check view_check_option
%type <str> opt_distinct opt_straight_join opt_cache match_option opt_separator opt_constraint opt_foreign_primary_unique
%type <expr> opt_like_escape
%type <hints> opt_hint_list hint_list
%type <hint> hint
%type <str> backend
%type <selectExprs> select_expression_list opt_select_expression_list opt_func_datetime_precision
%type <selectExpr> select_expression
%type <expr> expression range_expression list_expression
%type <tableExprs> opt_from table_references
%type <tableExpr> table_reference table_factor join_table
%type <tableNames> table_name_list delete_table_list
%type <str> inner_join outer_join natural_join opt_constraint_name
%type <tableName> table_name into_table_name opt_table_name delete_table_name
%type <procOrFuncName> proc_or_func_name
%type <aliasedTableName> aliased_table_name
%type <indexHints> index_hint_list
%type <scanMode> scan_mode_hint
%type <colIdents> part_list
%type <columnDetails> index_list
%type <expr> opt_where_expression
%type <expr> condition
%type <boolVal> boolean_value
%type <str> compare
%type <ins> insert_data
%type <expr> value value_expression num_val opt_interval
%type <expr> function_call_keyword function_call_nonkeyword function_call_generic function_call_conflict
%type <str> is_suffix
%type <colTuple> col_tuple
%type <exprs> expression_list range_expression_list list_expression_list
%type <values> tuple_list
%type <valTuple> row_tuple tuple_or_empty range_value_tuple list_value_tuple
%type <expr> tuple_expression
%type <subquery> subquery
%type <colName> column_name
%type <rowNumExpr> rownum
%type <whens> when_expression_list
%type <when> when_expression
%type <expr> opt_expression opt_else_expression
%type <exprs> opt_group_by
%type <expr> opt_having
%type <orderBy> opt_order_by order_list
%type <order> order
%type <str> opt_asc_desc
%type <limit> opt_limit
%type <str> opt_lock
%type <columns> ins_column_list opt_column_list column_list
%type <updateExprs> opt_on_dup
%type <updateExprs> update_list
%type <setExprs> set_list transaction_chars
%type <bytes> charset_or_character_set
%type <updateExpr> update_expression
%type <setExpr> set_expression transaction_char
%type <str> isolation_level
%type <str> opt_ignore opt_default
%type <byt> opt_exists opt_not_exists opt_temporary
%type <byt> opt_full opt_storage
%type <empty> non_add_drop_or_rename_operation opt_to opt_index
%type <bytes> reserved_keyword non_reserved_keyword
%type <bytes> unsupported_show_keyword
%type <colIdent> sql_id reserved_sql_id opt_partition_using
%type <columnDetail> index_id
%type <bytes> opt_index_length
%type <expr> charset_value col_alias opt_as_ci
%type <tableIdent> table_id reserved_table_id table_alias as_opt_id opt_partition_select
%type <empty> opt_as range_type list_type
%type <empty> force_eof ddl_force_eof
%type <str> charset opt_index_type algoritm_type lock_type
%type <str> variable_scope session_or_global
%type <convertType> convert_type
%type <columnType> column_type
%type <columnType> int_type decimal_type numeric_type time_type char_type spatial_type
%type <optVal> opt_length
%type <columnAttr> default_attribute opt_null null_attribute on_update_attribute auto_increment_attribute opt_column_comment column_comment_attribute
%type <str> opt_charset opt_collate opt_varying
%type <boolVal> opt_unsigned opt_zero_fill
%type <LengthScaleOption> opt_float_length opt_decimal_length
%type <expr> generated_as
%type <boolVal> opt_stored
%type <colKeyOpt> opt_column_key column_key
%type <strs> enum_values
%type <columnDefinition> column_definition
%type <indexDefinition> index_definition
%type <constraintDefinition> constraint_definition foreign_primary_unique_definition check_foreign_primary_unique_info
%type <constraintDetail> constraint_detail foreign_primary_unique_detail check_constraint
%type <constraintStateItem> constraint_state_item
%type <constraintStateItems> opt_constraint_state_items constraint_state_items
%type <indexOption> opt_index_option index_option
%type <indexOptions> opt_index_option_list index_option_list
%type <str> index_or_key opt_index_order opt_global indexes_or_keys
%type <str> opt_index_name opt_analyze_optimize opt_algorithm_option opt_opt_lockion
%type <TableSpec> table_spec table_column_list
%type <optLike> create_like
%type <optCreatePartition> opt_partition
%type <str> table_opt_value table_option_first_value storage_value database_opt_value
%type <tableOptions> table_option_list table_option table_option_def
%type <indexInfo> index_info
%type <indexColumn> index_column
%type <indexColumns> index_column_list key_part_list
%type <partDefs> partition_definitions
%type <partDef> partition_definition
%type <partSpec> partition_operation
%type <changeSpec> change_operation
%type <modifySpec> modify_operation
%type <kunpartDefs> range_partition_definitions list_partition_definitions
%type <kunpartDef> range_partition_definition list_partition_definition add_partition_definition
%type <triggerTime> trigger_time
%type <triggerEvent> trigger_event
%type <dbSpec> db_from_spec
%type <tableFromSpec> tb_from_spec
%type <showCondition> show_condition
%type <privileges> grant_privileges object_privilege_list
%type <object_privilege> object_privilege
%type <grantIdent> grant_ident
%type <principalDetail> grant_list
%type <principal> grant_user
%type <principalDetail> grant_role_list
%type <principal> grant_role
%type <principals> principal_list
%type <grant_option> opt_grant_option
%type <grant_option> opt_admin_option
%type <bytes> alter_object alter_object_type opt_column
%type <str> opt_ident ident
%type <ref> opt_references references
%type <onUpdateDelete> opt_on_update_delete
%type <refAction> fk_on_delete fk_on_update fk_reference_action
%type <optLocate> opt_locate
%type <str> user_or_role
%type <str> opt_definer opt_view_suid view_algorithm view_suid
%type <statement> explain_statement explainable_statement
%type <str> union_op insert_or_replace opt_explain_format opt_wild opt_order create_option opt_default_keyword opt_equal
%type <bytes> explain_synonyms
%type <columnAttrs> opt_column_attribute_list column_attribute_list
%type <columnAttr> column_attribute


%type <statement> create_package_stmt create_package_body_stmt create_function_stmt create_procedure_stmt alter_function_stmt drop_function_stmt drop_procedure_stmt alter_procedure_stmt drop_package_stmt alter_package_stmt
%type <str> opt_invoker_rights_clause is_or_as opt_name opt_ref opt_type_name opt_label_name opt_scope opt_savepoint opt_link_name immediate_batch immediate_deferred opt_local opt_year_day opt_month_second
%type <str> type_name opt_mode opt_constant opt_not_null opt_varray_prefix opt_declare opt_work quoted_string opt_write_clause wait_nowait constraint_constraints opt_specification
%type <str> only_write opt_name_quoted serializable_read_committed opt_pipelined pipelined_aggregate hash_range opt_for opt_percent opt_nocycle opt_debug opt_reuse_settings
%type <columnType> native_datatype_element
%type <colIdent> regular_id
%type <packageSpec> package_spec package_obj_spec_list
%type <pLDeclaration> pl_declaration
%type <packageBody> package_body package_obj_body_list
%type <pLPackageBody> pl_package_body
%type <procedureSpec> procedure_spec
%type <functionSpec> function_spec
%type <parameterList> parameter_list opt_parameter_list
%type <parameter> parameter
%type <typeSpec> opt_type_spec type_spec opt_cursor_return
%type <dataType> dataType
%type <variableDeclaration> variable_declaration
%type <parameterSpecType> opt_parameter_spec_type
%type <parameterSpec> parameter_spec
%type <parameterSpecList> opt_parameter_spec_list parameter_spec_list
%type <cursorDeclaration> cursor_declaration
%type <exceptionDeclaration> exception_declaration
%type <subTypeDeclaration> subtype_declaration
%type <subTypeRange> opt_subtype_range
%type <plsqlExpression> plsql_expression
%type <cursorExpression> cursor_expression
%type <logicalExpression> logical_expression
%type <typeDeclaration> type_declaration
%type <tableTypeDef> table_type_def
%type <tableIndex> opt_table_index
%type <varrayTypeDef> varray_type_def
%type <recordTypeDef> record_type_def
%type <fieldSpecList> field_spec_list
%type <fieldSpec> field_spec
%type <cursorTypeDef> cursor_type_def
%type <procedureDefinition> procedure_definition
%type <procedureBody> procedure_body
%type <callSpec> call_spec
%type <cSpec> c_spec
%type <block> block
%type <declareSpec> declare_spec
%type <body> body
%type <seqOfStatement> seq_of_statements
%type <loopStatement> loop_statement
%type <loopParam> loop_param
%type <pipeRowStatement> pipe_row_statement
%type <functionCallStatement> function_call_statement
%type <returnStatement> return_statement
%type <nullStatement> null_statement
%type <raiseStatement> raise_statement
%type <ifStatement> if_statement
%type <elsePart> else_part
%type <elseIfList> else_if_list
%type <elseIf> else_if
%type <goToStatement> go_to_statement
%type <exitStatement> exit_statement
%type <continueStatement> continue_statement
%type <assignmentStatement> assignment_statement
//%type <generalElement> general_element
//%type <generalElementPart> general_element_part
//%type <generalElementParList> opt_general_element_parts
//%type <bindVariable> bind_variable
//%type <str>  opt_charset_name bind_variable_prefix
%type <idExpressionList> id_expression_list
%type <idExpression> id_expression opt_id_expression
%type <exceptionHandlerList> exception_handler_list
%type <exceptionHandler> exception_handler
%type <defaultValuePart> opt_default_value_part
%type <pLStatement> pl_statement
%type <labelDeclaration> label_declaration
%type <plsqlExpressions> plsql_expressions opt_plsql_expressions
%type <expr> lower_bound upper_bound start_part opt_start_part
%type <caseStatement> case_statement
%type <simpleCaseStatement> opt_simple_case_statement
%type <searchCaseStatement> opt_search_case_statement
%type <caseElsePart> opt_case_else_part
%type <searchCaseWhenPart> search_case_when_part
%type <simpleCaseWhenPart> simple_case_when_part
%type <simpleCaseWhenPartList> simple_case_when_parts
%type <searchCaseWhenPartList> search_case_when_parts
%type <sqlStatement> sql_statement
%type <executeImmediate> execute_immediate
%type <usingReturn> using_return
%type <dynamicReturn> dynamic_return opt_dynamic_return
%type <intoUsing> into_using
%type <usingClause> using_clause
%type <usingElements> using_elements
%type <usingElement> using_element
%type <intoClause> into_clause opt_into_clause
%type <variableName> variable_name
%type <variableNameList> variable_names
%type <cursorManuStatement> cursor_manu_statement
%type <openForStatement> open_for_statement
%type <fetchStatement> fetch_statement
%type <openStatement> open_statement
%type <transCtrlStatement> trans_ctrl_statement
%type <setTransCommand> set_trans_command
%type <setConstCommand> set_const_command
%type <commitStatement> plsql_commit_statement
%type <rollbackStatement> plsql_rollback_statement
%type <savePointStatement> save_point_statement
%type <constraintName> constraint_name
%type <constraintNameList> constraint_names
%type <identifier> identifier
%type <dmlStatement> dml_tatement
%type <forAllStatement> forall_statement
%type <boundsClause> bounds_clause
%type <betweenBound> between_bound opt_between_bound
%type <indexName> index_name
%type <functionDefinition> function_definition
%type <functionBody> function_body
%type <funcReturnSuffixList> func_return_suffixs opt_func_return_suffixs
%type <funcReturnSuffix> func_return_suffix
%type <invokerRightsClause> invoker_rights_clause
%type <parallelEnableClause> parallel_enable_clause
%type <resultCacheClause> result_cache_clause
%type <deterministic> deterministic
%type <parenColumnList> paren_column_list
%type <streamingClause> stream_clause opt_stream_clause
%type <partitionByClause> partition_by_clause opt_partition_by_clause
%type <partitionExtension> partition_extension
%type <tableViewName> table_view_name
%type <tableViewNameList> table_view_names
%type <reliesOnPart> relies_on_part opt_relies_on_part
%type <hierarchicalQueryClause> opt_hierarchical_query_clause
%type <functionAlterBody> function_alter_body
%type <procedureAlterBody> procedure_alter_body
%type <packageAlterBody> package_alter_body
%type <compilerParameterList> compiler_parameter_list opt_compiler_parameter_list
%type <compilerParameter> compiler_parameter_clause


%start any_command

%%

/*
  Indentation of grammar rules:

rule: <-- starts at col 1
  {} <-- empty rule in one line
| rule1a rule1b rule1c <-- starts at col 3
  { <-- starts at col 3
    code <-- starts at col 5
  }
| rule2a rule2b <-- keep rule in same line with delimiter('|')
  {
    code
  }

  Also, please do not use any <TAB>, but spaces.
  Having a uniform indentation in this file helps
  code reviews, patches, merges, and make maintenance easier.

  indentation with 2 spaces
  Thanks.
*/

any_command:
  command opt_semicolon
  {
    setParseTree(yylex, $1)
  }

opt_semicolon:
/*empty*/ {}
| ';' {}

command:
  select_statement
  {
    $$ = $1
  }
| insert_statement
| update_statement
| delete_statement
| set_statement
| create_statement
| alter_statement
| rename_statement
| drop_statement
| truncate_statement
| show_statement
| use_statement
| begin_statement
| commit_statement
| rollback_statement
| explain_statement
| other_statement
| call_statement
| dcl_statement

dcl_statement:
  grant_statement
  {
    $$ = $1
  }
| revoke_statement
  {
    $$ = $1
  }
| drop_user_statement
  {
    $$ = $1
  }
| create_user_statement
  {
    $$ = $1
  }
| drop_role_statement
  {
    $$ = $1
  }
| create_role_statement
  {
    $$ = $1
  }

grant_statement:
  GRANT opt_comment grant_role TO grant_user opt_admin_option
  {
    $$ = &DCL{Action: GrantStr, Privileges: Privileges{&Privilege{Privilege: $3.Name()}}, GrantType: PrincipalRole, Principals: Principals{PrincipalType: PrincipalUser, PrincipalDetail: PrincipalDetail{$5}}, GrantOption: $6, Comments: Comments($2)}
  }
| GRANT opt_comment grant_privileges ON opt_table grant_ident TO principal_list opt_grant_option
  {
    $$ = &DCL{Action: GrantStr, Privileges: $3, GrantIdent: $6, Principals: $8, GrantOption: $9, Comments: Comments($2)}
  }
| GRANT opt_comment grant_privileges ON FUNCTION grant_ident TO principal_list opt_grant_option
  {
    $$ = &DCL{Action: GrantStr, Privileges: $3, GrantType: GrantTypeFunction, GrantIdent: $6, Principals: $8, GrantOption: $9, Comments: Comments($2)}
  }
| GRANT opt_comment grant_privileges ON PROCEDURE grant_ident TO principal_list opt_grant_option
  {
    $$ = &DCL{Action: GrantStr, Privileges: $3, GrantType: GrantTypeProcedure, GrantIdent: $6, Principals: $8, GrantOption: $9, Comments: Comments($2)}
  }

grant_privileges:
  ALL opt_privileges
  {
    $$ = Privileges{&Privilege{Privilege: "all"}}
  }
| object_privilege_list
  {
    $$ = $1
  }

opt_privileges:
  /* empty */ {}
| PRIVILEGES {}

object_privilege_list:
  object_privilege
  {
    $$ = Privileges{$1}
  }
| object_privilege_list ',' object_privilege
  {
    $$ = append($$, $3)
  }

object_privilege:
  SELECT opt_column_list
  {
    $$ = &Privilege{Privilege: "select", Columns: $2}
  }
| INSERT opt_column_list
  {
    $$ = &Privilege{Privilege: "insert", Columns: $2}
  }
| UPDATE opt_column_list
  {
    $$ = &Privilege{Privilege: "update", Columns: $2}
  }
| DELETE
  {
    $$ = &Privilege{Privilege: "delete"}
  }
| CREATE
  {
    $$ = &Privilege{Privilege: "create"}
  }
| DROP
  {
    $$ = &Privilege{Privilege: "drop"}
  }
| ALTER
  {
    $$ = &Privilege{Privilege: "alter"}
  }
| INDEX
  {
    $$ = &Privilege{Privilege: "index"}
  }
| CREATE ROUTINE
  {
    $$ = &Privilege{Privilege: "create routine"}
  }
| ALTER ROUTINE
  {
    $$ = &Privilege{Privilege: "alter routine"}
  }
| EXECUTE
  {
    $$ = &Privilege{Privilege: "execute"}
  }
| TRIGGER
  {
    $$ = &Privilege{Privilege: "trigger"}
  }
| USAGE
  {
    $$ = &Privilege{Privilege: "usage"}
  }
| CREATE USER
  {
    $$ = &Privilege{Privilege: "create user"}
  }
| CREATE VIEW
  {
    $$ = &Privilege{Privilege: "create view"}
  }

opt_column_list:
  /* Empty */
  {
    $$ = nil
  }
| openb ins_column_list closeb
  {
    $$ = $2
  }

opt_table:
  /* Empty */
| TABLE

grant_ident:
  '*' '.' '*'
   {
     $$ = &GrantIdent{WildcardKeyspaceName: true, WildcardTableName: true}
   }
| table_id '.' '*'
  {
    $$ = &GrantIdent{Schema: $1, WildcardTableName: true}
  }
| table_id '.' table_id
  {
    $$ = &GrantIdent{Schema : $1, Table: $3}
  }
| table_id
  {
    $$ = &GrantIdent{Table: $1}
  }

principal_list:
  ROLE grant_role_list
  {
    $$ = Principals{PrincipalType: PrincipalRole, PrincipalDetail: $2}
  }
| grant_list
  {
    $$ = Principals{PrincipalType: PrincipalUser, PrincipalDetail: $1}
  }

grant_list:
  grant_user
  {
    $$ = PrincipalDetail{$1}
  }
| grant_list ',' grant_user
  {
    $$ = append($$, $3)
  }

grant_user:
  sql_id
  {
    $$ = NewUser($1.String())
  }
| STRING
  {
    $$ = NewUser(string($1))
  }

grant_role_list:
  grant_role
  {
    $$ = PrincipalDetail{$1}
  }
| grant_role_list ',' grant_role
  {
    $$ = append($$, $3)
  }

grant_role:
  sql_id
  {
    $$ = NewRole($1.String())
  }
| STRING
  {
    $$ = NewRole(string($1))
  }

opt_grant_option:
  {
    $$ = false;
  }
| WITH GRANT OPTION
  {
    $$ = true;
  }

opt_admin_option:
  {
    $$ = false;
  }
| WITH ADMIN OPTION
  {
    $$ = true;
  }

revoke_statement:
  REVOKE opt_comment grant_role FROM grant_user
  {
    $$ = &DCL{Action: RevokeStr, Privileges: Privileges{&Privilege{Privilege: $3.Name()}}, GrantType: PrincipalRole, Principals: Principals{PrincipalType: PrincipalUser, PrincipalDetail: PrincipalDetail{$5}}, Comments: Comments($2)}
  }
| REVOKE opt_comment grant_privileges ON opt_table grant_ident FROM principal_list
  {
    $$ = &DCL{Action: RevokeStr, Privileges: $3, GrantIdent: $6, Principals: $8, Comments: Comments($2)}
  }
| REVOKE opt_comment grant_privileges ON FUNCTION grant_ident FROM principal_list
  {
    $$ = &DCL{Action: RevokeStr, Privileges: $3, GrantType: GrantTypeFunction, GrantIdent: $6, Principals: $8, Comments: Comments($2)}
  }
| REVOKE opt_comment grant_privileges ON PROCEDURE grant_ident FROM principal_list
  {
    $$ = &DCL{Action: RevokeStr, Privileges: $3, GrantType: GrantTypeProcedure, GrantIdent: $6, Principals: $8, Comments: Comments($2)}
  }
| REVOKE opt_comment GRANT OPTION ON opt_table grant_ident FROM principal_list
  {
    // though grant option is not a kind of privileges, but make it as one could simplify
    // the parsing & formatting of revoke statement
    privileges := Privileges{&Privilege{Privilege: "grant option"}}
    $$ = &DCL{Action: RevokeStr, Privileges: privileges, GrantIdent: $7, Principals: $9, Comments: Comments($2)}
  }
| REVOKE opt_comment GRANT OPTION ON FUNCTION grant_ident FROM principal_list
  {
    privileges := Privileges{&Privilege{Privilege: "grant option"}}
    $$ = &DCL{Action: RevokeStr, Privileges: privileges, GrantType: GrantTypeFunction, GrantIdent: $7, Principals: $9, Comments: Comments($2)}
  }
| REVOKE opt_comment GRANT OPTION ON PROCEDURE grant_ident FROM principal_list
  {
    privileges := Privileges{&Privilege{Privilege: "grant option"}}
    $$ = &DCL{Action: RevokeStr, Privileges: privileges, GrantType: GrantTypeProcedure, GrantIdent: $7, Principals: $9, Comments: Comments($2)}
  }

drop_user_statement:
  DROP USER opt_exists grant_list
  {
     var exists bool
     if $3 != 0 {
       exists = true
     }

    $$ = &DCL{Action: DropUserStr, Principals: Principals{PrincipalType: PrincipalUser, PrincipalDetail: $4}, IfExists: exists, GrantIdent: &GrantIdent{WildcardKeyspaceName: true, WildcardTableName: true}}
  }

create_user_statement:
  CREATE USER opt_not_exists grant_list
  {
    var not_exists bool
    if $3 != 0 {
        not_exists = true
    }

    $$ = &DCL{Action: CreateUserStr, Principals: Principals{PrincipalType: PrincipalUser, PrincipalDetail: $4}, IfExists: not_exists, GenPasswd: true, GrantIdent: &GrantIdent{WildcardKeyspaceName: true, WildcardTableName: true}}
  }

drop_role_statement:
  DROP ROLE opt_exists grant_role_list
  {
    var exists bool
    if $3 != 0 {
      exists = true
    }

    $$ = &DCL{Action: DropRoleStr, Principals: Principals{PrincipalType: PrincipalRole, PrincipalDetail: $4}, IfExists: exists, GrantIdent: &GrantIdent{WildcardKeyspaceName: true, WildcardTableName: true}}
  }

create_role_statement:
  CREATE ROLE opt_not_exists grant_role_list
  {
    var not_exists bool
    if $3 != 0 {
      not_exists = true
    }

    $$ = &DCL{Action: CreateRoleStr, Principals: Principals{PrincipalType: PrincipalRole, PrincipalDetail: $4}, IfExists: not_exists, GrantIdent: &GrantIdent{WildcardKeyspaceName: true, WildcardTableName: true}}
  }

select_statement:
  base_select opt_order_by opt_limit opt_lock
  {
    sel := $1.(*Select)
    sel.OrderBy = $2
    sel.Limit = $3
    sel.Lock = $4
    $$ = sel
  }
| union_lhs union_op union_rhs opt_order_by opt_limit opt_lock
  {
    $$ = &Union{Type: $2, Left: $1, Right: $3, OrderBy: $4, Limit: $5, Lock: $6}
  }
| SELECT opt_comment NEXT num_val FOR table_name
  {
    seqName := &ColName{Qualifier: TableName{Name: $6.Qualifier}, Name: NewColIdent($6.Name.String())}
    nextval := &FuncExpr{Name: NewColIdent("nextval"), Exprs: SelectExprs{&AliasedExpr{Expr: seqName, AsOpt: false}}}
    setHasSequenceUpdateNode(yylex, true)
    selectExpr := &AliasedExpr{Expr: nextval, AsOpt: false}
    dual := TableExprs{&AliasedTableExpr{Expr:TableName{Name: NewTableIdent("dual")}}}
    $$ = &Select{
      Comments: Comments($2),
      SelectExprs: SelectExprs{selectExpr},
      From: dual,
    }
  }

opt_nocycle:
  {
    $$ = ""
  }
| NOCYCLE
  {
    $$ = string($1)
  }

start_part:
  START WITH condition
  {
    $$ = $3
  }

opt_start_part:
  {
    $$ = nil
  }
| start_part
  {
    $$ = $1
  }

opt_hierarchical_query_clause:
  {
    $$ = nil
  }
| CONNECT BY opt_nocycle condition opt_start_part
  {
    $$ = &HierarchicalQueryClause{IsStartFirstly: false, NocycleOpt: $3, Condition: $4, StartPartOpt: $5}
  }
| start_part CONNECT BY opt_nocycle condition
  {
    $$ = &HierarchicalQueryClause{IsStartFirstly: true, StartPartOpt: $1, NocycleOpt: $4, Condition: $5}
  }

// base_select is an unparenthesized SELECT with no order by clause or beyond.
base_select:
  SELECT opt_comment opt_cache opt_distinct opt_straight_join opt_hint_list select_expression_list opt_into_clause opt_from opt_where_expression opt_hierarchical_query_clause opt_group_by opt_having
  {
    $$ = &Select{Comments: Comments($2), Cache: $3, Distinct: $4, Hints: $5, ExtraHints: $6, SelectExprs: $7, IntoClause: $8, From: $9, Where: NewWhere(TypWhere, $10), Hierarchical: $11, GroupBy: GroupBy($12), Having: NewWhere(TypHaving, $13)}
  }

union_lhs:
  select_statement
  {
    $$ = $1
  }
| openb select_statement closeb
  {
    $$ = &ParenSelect{Select: $2}
  }

union_rhs:
  base_select
  {
    $$ = $1
  }
| openb select_statement closeb
  {
    $$ = &ParenSelect{Select: $2}
  }


insert_statement:
  insert_or_replace opt_comment opt_ignore into_table_name insert_data opt_on_dup
  {
    // insert_data returns a *Insert pre-filled with Columns & Values
    ins := $5
    ins.Action = $1
    ins.Comments = $2
    ins.Ignore = $3
    ins.Table = $4
    ins.OnDup = OnDup($6)
    $$ = ins
  }
| insert_or_replace opt_comment opt_ignore into_table_name SET update_list opt_on_dup
  {
    cols := make(Columns, 0, len($6))
    vals := make(ValTuple, 0, len($7))
    for _, updateList := range $6 {
      cols = append(cols, updateList.Name.Name)
      vals = append(vals, updateList.Expr)
    }
    $$ = &Insert{Action: $1, Comments: Comments($2), Ignore: $3, Table: $4, Columns: cols, Rows: Values{vals}, OnDup: OnDup($7)}
  }

insert_or_replace:
  INSERT
  {
    $$ = InsertStr
  }
| REPLACE
  {
    $$ = ReplaceStr
  }

update_statement:
  UPDATE opt_comment table_references SET update_list opt_where_expression opt_order_by opt_limit
  {
    $$ = &Update{Comments: Comments($2), TableExprs: $3, Exprs: $5, Where: NewWhere(TypWhere, $6), OrderBy: $7, Limit: $8}
  }

delete_statement:
  DELETE opt_comment FROM table_name opt_where_expression opt_order_by opt_limit
  {
    $$ = &Delete{Comments: Comments($2), TableExprs:  TableExprs{&AliasedTableExpr{Expr:$4}}, Where: NewWhere(TypWhere, $5), OrderBy: $6, Limit: $7}
  }
| DELETE opt_comment FROM table_name_list USING table_references opt_where_expression
  {
    $$ = &Delete{Comments: Comments($2), Targets: $4, TableExprs: $6, Where: NewWhere(TypWhere, $7)}
  }
| DELETE opt_comment FROM delete_table_list USING table_references opt_where_expression
  {
    $$ = &Delete{Comments: Comments($2), Targets: $4, TableExprs: $6, Where: NewWhere(TypWhere, $7)}
  }
| DELETE opt_comment table_name_list from_or_using table_references opt_where_expression
  {
    $$ = &Delete{Comments: Comments($2), Targets: $3, TableExprs: $5, Where: NewWhere(TypWhere, $6)}
  }
| DELETE opt_comment delete_table_list from_or_using table_references opt_where_expression
  {
    $$ = &Delete{Comments: Comments($2), Targets: $3, TableExprs: $5, Where: NewWhere(TypWhere, $6)}
  }
| DELETE opt_comment FROM table_name opt_partition_select opt_where_expression opt_order_by opt_limit
  {
    $$ = &Delete{Comments: Comments($2), TableExprs:  TableExprs{&AliasedTableExpr{Expr:$4, Partition: $5}}, Where: NewWhere(TypWhere, $6), OrderBy: $7, Limit: $8}
  }

from_or_using:
  FROM {}
| USING {}

table_name_list:
  table_name
  {
    $$ = TableNames{$1}
  }
| table_name_list ',' table_name
  {
    $$ = append($$, $3)
  }

delete_table_list:
  delete_table_name
  {
    $$ = TableNames{$1}
  }
| delete_table_list ',' delete_table_name
  {
    $$ = append($$, $3)
  }

set_statement:
  SET opt_comment set_list
  {
    $$ = &Set{Comments: Comments($2), Exprs: $3}
   }
| SET opt_comment session_or_global set_list
  {
    $$ = &Set{Comments: Comments($2), Scope: $3, Exprs: $4}
  }
| SET opt_comment session_or_global TRANSACTION transaction_chars
  {
    $$ = &Set{Comments: Comments($2), Scope: $3, Exprs: $5}
  }
| SET opt_comment TRANSACTION transaction_chars
  {
    $$ = &Set{Comments: Comments($2), Exprs: $4}
  }

transaction_chars:
  transaction_char
  {
    $$ = SetExprs{$1}
  }
| transaction_chars ',' transaction_char
  {
    $$ = append($$, $3)
  }

transaction_char:
  ISOLATION LEVEL isolation_level
  {
    $$ = &SetExpr{Name: NewColIdent(TransactionIsolationLevelStr), Expr: NewStrVal([]byte($3))}
  }
| READ ONLY
  {
    $$ = &SetExpr{Name: NewColIdent(TransactionStr), Expr: NewStrVal([]byte(TxReadOnly))}
  }
| READ WRITE
  {
    $$ = &SetExpr{Name: NewColIdent(TransactionStr), Expr: NewStrVal([]byte(TxReadWrite))}
  }

isolation_level:
  REPEATABLE READ
  {
    $$ = IsolationLevelRepeatableRead
  }
| READ COMMITTED
  {
    $$ = IsolationLevelReadCommitted
  }
| READ UNCOMMITTED
  {
    $$ = IsolationLevelReadUncommitted
  }
| SERIALIZABLE
  {
    $$ = IsolationLevelSerializable
  }

session_or_global:
  LOCAL
  {
    $$ = SessionScope
  }
| SESSION
  {
    $$ = SessionScope
  }
| GLOBAL
  {
    $$ = GlobalScope
  }

opt_default_keyword:
  {
    $$ = ""
  }
| DEFAULT
  {
    $$ = string($1)
  }

opt_equal:
  {
    $$ = ""
  }
| '='
  {
    $$ = string($1)
  }

create_option:
  opt_default_keyword CHARACTER SET opt_equal database_opt_value
  {
    $$ = string($2) + " " + string($3) + " " + $5
  }
| opt_default_keyword CHARSET opt_equal database_opt_value
  {
    $$ = string($2) + " " + $4
  }
| opt_default_keyword COLLATE opt_equal database_opt_value
  {
    $$ = string($2) + " " + $4
  }
| create_option opt_default_keyword CHARACTER SET opt_equal database_opt_value
  {
    $$ = $1 + " " + string($3) + " " + string($4) + " " + $6
  }
| create_option opt_default_keyword CHARSET opt_equal database_opt_value
  {
    $$ = $1 + " " + string($3) + " " + $5
  }
| create_option opt_default_keyword COLLATE opt_equal database_opt_value
  {
    $$ = $1 + " " + string($3) + " " + $5
  }

database_opt_value:
  reserved_sql_id
  {
    $$ = $1.String()
  }
| STRING
  {
    $$ = "'" + string($1) + "'"
  }

create_statement:
  CREATE DATABASE opt_not_exists table_id
  {
    var not_exists bool
    if $3 != 0 {
      not_exists = true
    }
    tmpSchemaSpec := &SchemaSpec{SchemaName:$4}
    $$ = &DDL{Action: CreateStr, IfNotExists: not_exists, SchemaSpec: tmpSchemaSpec, DDLType: SchemaDDL}
  }
| CREATE SCHEMA opt_not_exists table_id
  {
    var not_exists bool
    if $3 != 0 {
      not_exists = true
    }
    tmpSchemaSpec := &SchemaSpec{SchemaName:$4}
    $$ = &DDL{Action: CreateStr, IfNotExists: not_exists, SchemaSpec: tmpSchemaSpec, DDLType: SchemaDDL}
  }
| CREATE DATABASE opt_not_exists table_id create_option
  {
    var not_exists bool
    if $3 != 0 {
      not_exists = true
    }
    tmpSchemaSpec := &SchemaSpec{SchemaName:$4, Options: $5}
    $$ = &DDL{Action: CreateStr, IfNotExists: not_exists, SchemaSpec: tmpSchemaSpec, DDLType: SchemaDDL}
  }
| CREATE SCHEMA opt_not_exists table_id create_option
  {
    var not_exists bool
    if $3 != 0 {
      not_exists = true
    }
    tmpSchemaSpec := &SchemaSpec{SchemaName:$4, Options: $5}
    $$ = &DDL{Action: CreateStr, IfNotExists: not_exists, SchemaSpec: tmpSchemaSpec, DDLType: SchemaDDL}
  }
| create_table_prefix table_spec opt_partition opt_locate
  {
    $1.TableSpec = $2
    $1.OptCreatePartition = $3
    $1.OptLocate = $4
    $$ = $1
    err := CheckPartitionValid($1)
    if err != nil {
        yylex.Error(err.Error())
        return 1
    }
  }
| create_table_prefix create_like
  {
    // Create table [name] like [name]
    $1.OptLike = $2
    $$ = $1
  }
| CREATE GLOBAL INDEX ID opt_index_type ON table_name key_part_list opt_index_option opt_algorithm_option opt_opt_lockion
  {
     tempInfo := &IndexInfo{Type: "index", Name: NewTableIdent(string($4))}
     tempIdxDef := &IndexDefinition{Info: tempInfo, Using: $5, TblName: $7, Columns: $8}
     tempIndexSpec := &IndexSpec{IdxDef: tempIdxDef, Global: true}
     $$ = &DDL{Action: CreateStr, Table: $7, NewName:$7, IndexSpec: tempIndexSpec, DDLType: IndexDDL}
  }
| CREATE GLOBAL UNIQUE INDEX ID opt_index_type ON table_name key_part_list opt_index_option opt_algorithm_option opt_opt_lockion
  {
     tempInfo := &IndexInfo{Type: "unique index", Name: NewTableIdent(string($5)), Unique: true}
     tempIdxDef := &IndexDefinition{Info: tempInfo, Using: $6, TblName: $8, Columns: $9}
     tempIndexSpec := &IndexSpec{IdxDef: tempIdxDef, Global: true}
     $$ = &DDL{Action: CreateStr, Table: $8, NewName:$8, IndexSpec: tempIndexSpec, DDLType: IndexDDL}
  }
| CREATE LOCAL opt_constraint INDEX ID opt_index_type ON table_name key_part_list opt_index_option opt_algorithm_option opt_opt_lockion
  {
     var tempInfo *IndexInfo
     indexName := NewTableIdent(string($5))
     if strings.EqualFold($3, "unique") {
     	tempInfo = &IndexInfo{Type: "unique index", Name: indexName, Unique: true}
     } else if strings.EqualFold($3, "spatial") {
     	tempInfo = &IndexInfo{Type: "spatial index", Name: indexName, Spatial: true}
     } else if strings.EqualFold($3, "fulltext") {
        tempInfo = &IndexInfo{Type: "fulltext index", Name: indexName, Fulltext: true}
     } else {
     	tempInfo = &IndexInfo{Type: "index", Name: indexName}
     }
     tempIdxDef := &IndexDefinition{Info: tempInfo, Using: $6, TblName: $8, Columns: $9, IndexOpts: []*IndexOption{$10}, AlgorithmOption: $11, LockOption: $12}
     tempIndexSpec := &IndexSpec{IdxDef: tempIdxDef, Global: false}
     $$ = &DDL{Action: CreateStr, Table: $8, NewName:$8, IndexSpec: tempIndexSpec, DDLType: IndexDDL}
  }
| CREATE opt_constraint INDEX ID opt_index_type ON table_name key_part_list opt_index_option opt_algorithm_option opt_opt_lockion
  {
     var tempInfo *IndexInfo
     indexName := NewTableIdent(string($4))
     if strings.EqualFold($2, "unique") {
        tempInfo = &IndexInfo{Type: "unique index", Name: indexName, Unique: true}
     } else if strings.EqualFold($2, "spatial") {
        tempInfo = &IndexInfo{Type: "spatial index", Name: indexName, Spatial: true}
     } else if strings.EqualFold($2, "fulltext") {
        tempInfo = &IndexInfo{Type: "fulltext index", Name: indexName, Fulltext: true}
     } else {
        tempInfo = &IndexInfo{Type: "index", Name: indexName}
     }
     tempIdxDef := &IndexDefinition{Info: tempInfo, Using: $5, TblName: $7, Columns: $8, IndexOpts: []*IndexOption{$9}, AlgorithmOption: $10, LockOption: $11}
     tempIndexSpec := &IndexSpec{IdxDef: tempIdxDef, Global: false}
     $$ = &DDL{Action: CreateStr, Table: $7, NewName:$7, IndexSpec: tempIndexSpec, DDLType: IndexDDL}
  }
| CREATE opt_definer opt_view_suid VIEW opt_comment opt_not_exists table_name AS select_statement opt_with_check
  {
    var not_exists bool
    if $6 != 0 {
        not_exists = true
    }
    tmpViewSpec := &ViewSpec{ViewName:$7.ToViewName(), Select: $9, WithCheckOption: $10}
    $$ = &DDL{Action: CreateStr, Comments: Comments($5), IfNotExists: not_exists, NewName: $7.ToViewName(), ViewSpec:tmpViewSpec, DDLType: ViewDDL}
  }
| CREATE view_algorithm opt_definer opt_view_suid VIEW opt_comment opt_not_exists table_name AS select_statement opt_with_check
  {
    var not_exists bool
    if $7 != 0 {
        not_exists = true
    }
    tmpViewSpec := &ViewSpec{ViewName:$8.ToViewName(), Select: $10, WithCheckOption: $11}
    $$ = &DDL{Action: CreateStr, Comments: Comments($6), IfNotExists: not_exists, NewName: $8.ToViewName(), ViewSpec:tmpViewSpec, DDLType: ViewDDL}
  }
| CREATE OR REPLACE opt_definer opt_view_suid VIEW opt_comment table_name AS select_statement opt_with_check
  {
    tmpViewSpec := &ViewSpec{ViewName:$8.ToViewName(), Select: $10, WithCheckOption: $11}
    $$ = &DDL{Action: CreateStr, Comments: Comments($7), NewName: $8.ToViewName(), ViewSpec:tmpViewSpec, DDLType: ViewDDL, CreateOrReplace: true}
  }
| CREATE OR REPLACE view_algorithm opt_definer opt_view_suid VIEW opt_comment table_name AS select_statement opt_with_check
  {
    tmpViewSpec := &ViewSpec{ViewName:$9.ToViewName(), Select: $11, WithCheckOption: $12}
    $$ = &DDL{Action: CreateStr, Comments: Comments($8), NewName: $9.ToViewName(), ViewSpec:tmpViewSpec, DDLType: ViewDDL, CreateOrReplace: true}
  }
| create_sequence_statement
  {
    $$ = $1
  }
| CREATE opt_definer TRIGGER opt_comment opt_not_exists table_name trigger_time trigger_event ON table_name FOR EACH ROW ddl_force_eof
  {
    var not_exists bool
    if $5 != 0 {
        not_exists = true
    }
    tmpTriggerSpec := &TriggerSpec{TriggerName:$6, TriggerTime:$7, TriggerEvent:$8, TriggerTarget:$10}
    $$ = &DDL{Action: CreateStr, NewName:$6, Comments: Comments($4), IfNotExists:not_exists, TriggerSpec:tmpTriggerSpec, DDLType: TriggerDDL}
  }
| CREATE OR REPLACE opt_definer TRIGGER opt_comment opt_not_exists table_name trigger_time trigger_event ON table_name FOR EACH ROW ddl_force_eof
  {
    var not_exists bool
    if $7 != 0 {
        not_exists = true
    }
    tmpTriggerSpec := &TriggerSpec{TriggerName:$8, TriggerTime:$9, TriggerEvent:$10, TriggerTarget:$12}
    $$ = &DDL{Action: CreateStr, NewName:$8, Comments: Comments($6), IfNotExists:not_exists, TriggerSpec:tmpTriggerSpec, DDLType: TriggerDDL, CreateOrReplace: true}
  }
| CREATE SERVER opt_not_exists table_id ddl_force_eof
  {
    var not_exists bool
    if $3 != 0 {
      not_exists = true
    }
    tmpServerSpec := &ServerSpec{ServerName:$4}
    $$ = &DDL{Action: CreateStr, IfNotExists: not_exists, ServerSpec: tmpServerSpec, DDLType: ServerDDL}
  }
| CREATE SYNONYM opt_not_exists table_name FOR table_name
  {
    var not_exists bool
    if $3 != 0 {
      not_exists = true
    }
    tmpSynonymSpec := &SynonymSpec{SynonymName: $4, TargetName: $6}
    $$ = &DDL{Action: CreateStr, IfNotExists: not_exists, SynonymSpec: tmpSynonymSpec, DDLType: SynonymDDL}
  }
| create_package_stmt
  {
    $$ = $1
  }
| create_package_body_stmt
  {
    $$ = $1
  }
| create_function_stmt
  {
    $$ = $1
  }
| create_procedure_stmt
  {
    $$ = $1
  }

opt_with_check:
  {
    $$ = ""
  }
| WITH view_check_option CHECK OPTION
  {
    $$ = string($1) + " " + $2 + " " + string($3) + " " + string($4)
  }

view_check_option:
  LOCAL
  {
    $$ = string($1)
  }
| CASCADED
  {
    $$ = string($1)
  }

trigger_time:
  BEFORE
  {
    $$ = string($1)
  }
| AFTER
  {
    $$ = string($1)
  }

trigger_event:
  INSERT
  {
    $$ = string($1)
  }
| UPDATE
  {
    $$ = string($1)
  }
| DELETE
  {
    $$ = string($1)
  }

create_sequence_statement:
  CREATE SEQUENCE opt_not_exists table_name table_option_list
  {
    var not_exists bool
    if $3 != 0 {
        not_exists = true
    }

    tmpSeqSpec := &SequenceSpec {
      FullName: $4,
      StartIndex: NewIntVal([]byte("1")),
      CacheSize: NewIntVal([]byte("1000")),
      Options: $5,
    }
    $$ = &DDL{Action: CreateStr, NewName:$4, IfNotExists:not_exists, SequenceSpec:tmpSeqSpec, DDLType: SequenceDDL}
  }
| CREATE SEQUENCE opt_not_exists table_name START WITH INTEGRAL
  {
    var not_exists bool
    if $3 != 0 {
        not_exists = true
    }

    tmpSeqSpec := &SequenceSpec{FullName:$4, StartIndex:NewIntVal($7), CacheSize:NewIntVal([]byte("100"))}
    $$ = &DDL{Action: CreateStr, NewName:$4, IfNotExists:not_exists, SequenceSpec:tmpSeqSpec, DDLType: SequenceDDL}
  }
| CREATE SEQUENCE opt_not_exists table_name CACHE INTEGRAL
  {
    var not_exists bool
    if $3 != 0 {
        not_exists = true
    }
    tmpSeqSpec := &SequenceSpec{FullName:$4, StartIndex:NewIntVal([]byte("0")), CacheSize:NewIntVal($6)}
    $$ = &DDL{Action: CreateStr, NewName:$4, IfNotExists:not_exists, SequenceSpec:tmpSeqSpec, DDLType: SequenceDDL}
  }
| CREATE SEQUENCE opt_not_exists table_name START WITH INTEGRAL CACHE INTEGRAL
  {
    var not_exists bool
    if $3 != 0 {
        not_exists = true
    }
    tmpSeqSpec := &SequenceSpec{FullName:$4, StartIndex:NewIntVal($7), CacheSize:NewIntVal($9)}
    $$ = &DDL{Action: CreateStr, NewName:$4, IfNotExists:not_exists, SequenceSpec:tmpSeqSpec, DDLType: SequenceDDL}
  }

create_table_prefix:
  CREATE opt_temporary TABLE opt_not_exists table_name
  {
    var temporary_exist bool
    if $2 != 0 {
        temporary_exist = true
    }
    var not_exists bool
    if $4 != 0 {
        not_exists = true
    }
    $$ = &DDL{Action: CreateStr, IfTemporary: temporary_exist, IfNotExists: not_exists, NewName: $5, DDLType: TableDDL}
    setDDL(yylex, $$)
  }

table_spec:
  '(' table_column_list ')' table_option_list
  {
    $$ = $2
    $$.Options = $4
  }

opt_partition:
  {
    $$ = nil
  }
| PARTITION BY HASH '(' ID ')' opt_partition_using
  {
   tmpCloumn := []ColIdent{}
   tmpCloumn = append(tmpCloumn, NewColIdent(string($5)))
   $$ = &OptCreatePartition{Column: tmpCloumn, ShardFunc: $7, OptUsing: NewColIdent(HashPartition)}
  }
| PARTITION BY REFERENCE '(' ID ')'
  {
    tmpCloumn := []ColIdent{}
    tmpCloumn = append(tmpCloumn, NewColIdent(string($5)))
    $$ = &OptCreatePartition{Column: tmpCloumn, OptUsing: NewColIdent(ReferencePartition)}
  }
| PARTITION BY REPLICATION
  {
    $$ = &OptCreatePartition{OptUsing: NewColIdent(ReplicationPartition)}
  }
| PARTITION BY range_type '(' part_list ')' opt_interval '(' range_partition_definitions ')'
  {
    tmpUsing := NewColIdent(RangePartition)
    if $7 != nil {
        tmpUsing = NewColIdent(IntervalPartition)
    }
    $$ = &OptCreatePartition{Column: $5, OptUsing: tmpUsing, OptInterval: $7, Definitions: $9}
  }
| PARTITION BY list_type '(' ID ')' '(' list_partition_definitions ')'
  {
    tmpCloumn := []ColIdent{}
    tmpCloumn = append(tmpCloumn, NewColIdent(string($5)))
    $$ = &OptCreatePartition{Column: tmpCloumn, OptUsing: NewColIdent(ListPartition), Definitions: $8}
  }

range_type:
  RANGE
  { $$ = struct{}{} }
| RANGE COLUMNS
  { $$ = struct{}{} }

list_type:
  LIST
  { $$ = struct{}{} }
| LIST COLUMNS
  { $$ = struct{}{} }


opt_locate:
  {
    $$ = nil
  }
| LOCATE '(' ID ')'
  {
    $$ = &OptLocate{Location: string($3)}
  }

create_like:
  LIKE table_name
  {
    $$ = &OptLike{LikeTable: $2}
  }
| openb LIKE table_name closeb
  {
    $$ = &OptLike{LikeTable: $3}
  }

table_column_list:
  column_definition
  {
    $$ = &TableSpec{allChecks: true}
    $$.AddColumn($1)
  }
| table_column_list ',' column_definition
  {
    $$.AddColumn($3)
  }
| table_column_list ',' index_definition
  {
    $$.AddIndex($3)
  }
| table_column_list ',' constraint_definition
  {
    $$.AddConstraint($3)
  }

opt_column_attribute_list:
  /* empty */
  {
    $$ = nil
  }
| column_attribute_list
  {
    $$ = $1
  }

column_attribute_list:
  column_attribute_list column_attribute
  {
    $$ = append($$, $2)
  }
| column_attribute
  {
    $$ = ColumnAttrList{$1}
  }

column_attribute:
  default_attribute
  {
    $$ = $1
  }
| null_attribute
  {
    $$ = $1
  }
| on_update_attribute
  {
    $$ = $1
  }
| auto_increment_attribute
  {
    $$ = $1
  }
| column_key
  {
    $$ = $1
  }
| column_comment_attribute
  {
    $$ = $1
  }
| opt_constraint_name check_constraint
  {
    $$ = &ColumnConstraint{Name: $1, Detail: $2}
  }
| constraint_state_item
  {
    $$ = $1
  }

column_definition:
  sql_id column_type opt_column_attribute_list opt_references
  {
    /* FK references is ignored */
    if $3 != nil {
      $2.Generated = BoolVal(false)
      $2.Default, _ = $3.parseDefault()
      $2.NotNull, _ = $3.parseNull()
      $2.OnUpdate, _ = $3.parseOnUpdate()
      $2.Autoincrement, _ = $3.parseAutoInc()
      $2.KeyOpt, _ = $3.parseKeyOpt()
      $2.Comment, _ = $3.parseComment()
      $2.Constraint, _ = $3.parseConstraint()
    }
    $$ = &ColumnDefinition{Name: $1, Type: $2}
  }
| sql_id column_type generated_as opt_stored opt_null opt_column_key opt_column_comment
  {
    $2.Generated = BoolVal(true)
    $2.As = $3
    $2.Stored = $4
    $2.NotNull = $5.(*NullAttr).GetAttr()
    $2.KeyOpt = $6
    if $7 != nil {
      $2.Comment = $7.(*ColumnComment).GetAttr()
    }
    $$ = &ColumnDefinition{Name: $1, Type: $2}
  }

column_type:
  numeric_type opt_unsigned opt_zero_fill
  {
    $$ = $1
    $$.Unsigned = $2
    $$.Zerofill = $3
  }
| char_type
| time_type
| spatial_type
  {
    $$.Spatial = BoolVal(true)
  }

numeric_type:
  int_type opt_length
  {
    $$ = $1
    $$.Length = $2
  }
| decimal_type
  {
    $$ = $1
  }

int_type:
  BIT
  {
    $$ = ColumnType{Type: string($1)}
  }
| BOOL
  {
    $$ = ColumnType{Type: string($1)}
  }
| BOOLEAN
  {
    $$ = ColumnType{Type: string($1)}
  }
| TINYINT
  {
    $$ = ColumnType{Type: string($1)}
  }
| SMALLINT
  {
    $$ = ColumnType{Type: string($1)}
  }
| MEDIUMINT
  {
    $$ = ColumnType{Type: string($1)}
  }
| INT
  {
    $$ = ColumnType{Type: string($1)}
  }
| INTEGER
  {
    $$ = ColumnType{Type: string($1)}
  }
| BIGINT
  {
    $$ = ColumnType{Type: string($1)}
  }

decimal_type:
REAL opt_float_length
  {
    $$ = ColumnType{Type: string($1)}
    $$.Length = $2.Length
    $$.Scale = $2.Scale
  }
| DOUBLE opt_float_length
  {
    $$ = ColumnType{Type: string($1)}
    $$.Length = $2.Length
    $$.Scale = $2.Scale
  }
| FLOAT_TYPE opt_decimal_length
  {
    $$ = ColumnType{Type: string($1)}
    $$.Length = $2.Length
    $$.Scale = $2.Scale
  }
| DECIMAL opt_decimal_length
  {
    $$ = ColumnType{Type: string($1)}
    $$.Length = $2.Length
    $$.Scale = $2.Scale
  }
| NUMERIC opt_decimal_length
  {
    $$ = ColumnType{Type: string($1)}
    $$.Length = $2.Length
    $$.Scale = $2.Scale
  }

time_type:
  DATE
  {
    $$ = ColumnType{Type: string($1)}
  }
| TIME opt_length
  {
    $$ = ColumnType{Type: string($1), Length: $2}
  }
| TIMESTAMP opt_length
  {
    $$ = ColumnType{Type: string($1), Length: $2}
  }
| DATETIME opt_length
  {
    $$ = ColumnType{Type: string($1), Length: $2}
  }
| YEAR opt_length
  {
    $$ = ColumnType{Type: string($1), Length: $2}
  }


char_type:
  CHAR opt_length opt_charset opt_collate
  {
    $$ = ColumnType{Type: string($1), Length: $2, Charset: $3, Collate: $4}
  }
| VARCHAR opt_length opt_charset opt_collate
  {
    $$ = ColumnType{Type: string($1), Length: $2, Charset: $3, Collate: $4}
  }
| CHARACTER opt_varying opt_length opt_charset opt_collate
  {
    tmpType := string($1)
    if $2 != "" {
       tmpType += " " + string($2)
    }
    $$ = ColumnType{Type: tmpType, Length: $3, Charset: $4, Collate: $5}
  }
| BINARY opt_length
  {
    $$ = ColumnType{Type: string($1), Length: $2}
  }
| VARBINARY opt_length
  {
    $$ = ColumnType{Type: string($1), Length: $2}
  }
| TEXT opt_length opt_charset opt_collate
  {
    $$ = ColumnType{Type: string($1), Length: $2, Charset: $3, Collate: $4}
  }
| TINYTEXT opt_charset opt_collate
  {
    $$ = ColumnType{Type: string($1), Charset: $2, Collate: $3}
  }
| MEDIUMTEXT opt_charset opt_collate
  {
    $$ = ColumnType{Type: string($1), Charset: $2, Collate: $3}
  }
| LONGTEXT opt_charset opt_collate
  {
    $$ = ColumnType{Type: string($1), Charset: $2, Collate: $3}
  }
| CLOB
  {
    $$ = ColumnType{Type: "longtext"}
  }
| BLOB
  {
    $$ = ColumnType{Type: string($1)}
  }
| TINYBLOB
  {
    $$ = ColumnType{Type: string($1)}
  }
| MEDIUMBLOB
  {
    $$ = ColumnType{Type: string($1)}
  }
| LONGBLOB
  {
    $$ = ColumnType{Type: string($1)}
  }
| JSON
  {
    $$ = ColumnType{Type: string($1)}
  }
| ENUM '(' enum_values ')' opt_charset opt_collate
  {
    $$ = ColumnType{Type: string($1), EnumValues: $3, Charset: $5, Collate: $6}
  }

spatial_type:
  GEOMETRY
  {
    $$ = ColumnType{Type: string($1)}
  }
| POINT
  {
    $$ = ColumnType{Type: string($1)}
  }
| LINESTRING
  {
    $$ = ColumnType{Type: string($1)}
  }
| POLYGON
  {
    $$ = ColumnType{Type: string($1)}
  }
| GEOMETRYCOLLECTION
  {
    $$ = ColumnType{Type: string($1)}
  }
| MULTIPOINT
  {
    $$ = ColumnType{Type: string($1)}
  }
| MULTILINESTRING
  {
    $$ = ColumnType{Type: string($1)}
  }
| MULTIPOLYGON
  {
    $$ = ColumnType{Type: string($1)}
  }

enum_values:
  STRING
  {
    $$ = make([]string, 0, 4)
    $$ = append($$, "'" + string($1) + "'")
  }
| enum_values ',' STRING
  {
    $$ = append($1, "'" + string($3) + "'")
  }

opt_length:
  {
    $$ = nil
  }
| '(' INTEGRAL ')'
  {
    $$ = NewIntVal($2)
  }

opt_float_length:
  {
    $$ = LengthScaleOption{}
  }
| '(' INTEGRAL ',' INTEGRAL ')'
  {
    $$ = LengthScaleOption{
        Length: NewIntVal($2),
        Scale: NewIntVal($4),
    }
  }

opt_decimal_length:
  {
    $$ = LengthScaleOption{}
  }
| '(' INTEGRAL ')'
  {
    $$ = LengthScaleOption{
        Length: NewIntVal($2),
    }
  }
| '(' INTEGRAL ',' INTEGRAL ')'
  {
    $$ = LengthScaleOption{
        Length: NewIntVal($2),
        Scale: NewIntVal($4),
    }
  }

opt_unsigned:
  {
    $$ = BoolVal(false)
  }
| UNSIGNED
  {
    $$ = BoolVal(true)
  }

opt_zero_fill:
  {
    $$ = BoolVal(false)
  }
| ZEROFILL
  {
    $$ = BoolVal(true)
  }

generated_as:
  AS '(' expression ')'
  {
    $$ = $3
  }
| GENERATED ALWAYS AS '(' expression ')'
  {
    $$ = $5
  }

opt_stored:
  {
    $$ = BoolVal(false)
  }
| VIRTUAL
  {
    $$ = BoolVal(false)
  }
| STORED
  {
    $$ = BoolVal(true)
  }

// Null opt returns false to mean NULL (i.e. the default) and true for NOT NULL
opt_null:
  {
    $$ = &NullAttr{attr: BoolVal(false)}
  }
| null_attribute
  {
    $$ = $1
  }

null_attribute:
  NULL
  {
    $$ = &NullAttr{attr: BoolVal(false)}
  }
| NOT NULL
  {
    $$ = &NullAttr{attr: BoolVal(true)}
  }

default_attribute:
  DEFAULT STRING
  {
    $$ = &DefaultAttr{attr: NewStrVal($2)}
  }
| DEFAULT INTEGRAL
  {
    $$ = &DefaultAttr{attr: NewIntVal($2)}
  }
| DEFAULT '-' INTEGRAL
  {
    $$ = &DefaultAttr{attr: NewIntVal(append([]byte("-"), $3...))}
  }
| DEFAULT FLOAT
  {
    $$ = &DefaultAttr{attr: NewFloatVal($2)}
  }
| DEFAULT '-' FLOAT
  {
    $$ = &DefaultAttr{attr: NewFloatVal(append([]byte("-"), $3...))}
  }
| DEFAULT NULL
  {
    $$ = &DefaultAttr{attr: NewValArg($2)}
  }
| DEFAULT boolean_value
  {
    $$ = &DefaultAttr{attr: NewSQLBoolVal($2)}
  }
| DEFAULT CURRENT_TIMESTAMP
  {
    $$ = &DefaultAttr{attr: NewValArg($2)}
  }
| DEFAULT CURRENT_TIMESTAMP '(' ')'
  {
    $$ = &DefaultAttr{attr: NewValArg($2)}
  }
| DEFAULT BIT_LITERAL
  {
    $$ = &DefaultAttr{attr: NewBitVal($2)}
  }

on_update_attribute:
  ON UPDATE CURRENT_TIMESTAMP
  {
    $$ = &OnUpdateAttr{attr: NewValArg($3)}
  }
| ON UPDATE CURRENT_TIMESTAMP '(' ')'
  {
    $$ = &OnUpdateAttr{attr: NewValArg($3)}
  }


auto_increment_attribute:
  AUTO_INCREMENT
  {
    $$ = &AutoIncAttr{attr: BoolVal(true)}
  }

opt_charset:
  {
    $$ = ""
  }
| CHARACTER SET ID
  {
    $$ = string($3)
  }
| CHARACTER SET BINARY
  {
    $$ = string($3)
  }

opt_collate:
  {
    $$ = ""
  }
| COLLATE ID
  {
    $$ = string($2)
  }
| COLLATE STRING
  {
    $$ = string($2)
  }

opt_varying:
  {
    $$ = ""
  }
| VARYING
  {
    $$ = string($1)
  }

opt_column_key:
  {
    $$ = ColKeyNone
  }
| column_key
  {
    $$ = $1
  }

column_key:
  PRIMARY KEY
  {
    $$ = ColKeyPrimary
  }
| KEY
  {
    $$ = ColKey
  }
| UNIQUE KEY
  {
    $$ = ColKeyUniqueKey
  }
| UNIQUE
  {
    $$ = ColKeyUnique
  }

opt_column_comment:
  {
    $$ = nil
  }
| column_comment_attribute

column_comment_attribute:
  COMMENT_KEYWORD STRING
  {
    $$ = &ColumnComment{attr: NewStrVal($2)}
  }

index_definition:
  index_info opt_index_type key_part_list opt_index_option_list
  {
    $$ = &IndexDefinition{Info: $1, Using: $2, Columns: $3, IndexOpts: $4}
  }

index_info:
  PRIMARY KEY opt_index_name
  {
    $$ = &IndexInfo{Type: string($1) + " " + string($2), Name: NewTableIdent("PRIMARY"), Primary: true, Unique: true}
  }
| CONSTRAINT ID PRIMARY KEY opt_index_name
  {
    $$ = &IndexInfo{Type: string($3) + " " + string($4), Name: NewTableIdent("PRIMARY"), Primary: true, Unique: true}
  }
| SPATIAL index_or_key opt_index_name
  {
    $$ = &IndexInfo{Type: string($1) + " " + string($2), Name: NewTableIdent($3), Spatial: true, Unique: false}
  }
| UNIQUE index_or_key opt_index_name
  {
    $$ = &IndexInfo{Type: string($1) + " " + string($2), Name: NewTableIdent($3), Unique: true}
  }
| UNIQUE opt_index_name
  {
    $$ = &IndexInfo{Type: string($1), Name: NewTableIdent($2), Unique: true}
  }
| CONSTRAINT ID UNIQUE index_or_key opt_index_name
  {
    $$ = &IndexInfo{Type: string($3) + " " + string($4), Name: NewTableIdent($5), Unique: true}
  }
| CONSTRAINT ID UNIQUE opt_index_name
  {
    $$ = &IndexInfo{Type: string($3), Name: NewTableIdent($4), Unique: true}
  }
| FULLTEXT index_or_key opt_index_name
  {
    $$ = &IndexInfo{Type: string($1) + " " + string($2), Name: NewTableIdent($3), Unique: true}
  }
| FULLTEXT opt_index_name
  {
    $$ = &IndexInfo{Type: string($1), Name: NewTableIdent($2), Unique: true}
  }
| index_or_key opt_index_name
  {
    $$ = &IndexInfo{Type: string($1), Name: NewTableIdent($2), Unique: false}
  }

index_or_key:
  INDEX
  {
    $$ = string($1)
  }
| KEY
  {
    $$ = string($1)
  }

indexes_or_keys:
  INDEX
  {
    $$ = string($1)
  }
| INDEXES
  {
    $$ = string($1)
  }
| KEYS
  {
    $$ = string($1)
  }

opt_index_name:
  {
    $$ = ""
  }
| ID
  {
    $$ = string($1)
  }

index_column_list:
  index_column
  {
    $$ = []*IndexColumn{$1}
  }
| index_column_list ',' index_column
  {
    $$ = append($$, $3)
  }

index_column:
  sql_id opt_length opt_index_order
  {
      $$ = &IndexColumn{Column: $1, Length: $2, Order: $3}
  }

opt_index_order:
   {
     $$ = ""
   }
 | ASC
   {
     $$ = string($1)
   }
 | DESC
   {
     $$ = string($1)
   }

 constraint_definition:
   CONSTRAINT ID constraint_detail opt_constraint_state_items
   {
     $$ = &ConstraintDefinition{Name: string($2), Detail: $3, State: mergeConstraintState($4)}
   }
 | constraint_detail opt_constraint_state_items
   {
    $$ = &ConstraintDefinition{Detail: $1, State: mergeConstraintState($2)}
   }

opt_constraint_state_items:
  {}
| constraint_state_items

constraint_state_items:
  constraint_state_item
  {
    $$ = []*ConstraintStateItem{$1}
  }
| constraint_state_items constraint_state_item
  {
    $$ = append($$, $2)
  }

constraint_state_item:
  ENABLE
  {
    $$ = &ConstraintStateItem{Typ: TypEnable, IsEnabled: true}
  }
| DISABLE
  {
    $$ = &ConstraintStateItem{Typ: TypEnable}
  }
| DEFERRABLE
  {
    $$ = &ConstraintStateItem{}
  }
| NOT DEFERRABLE
  {
    $$ = &ConstraintStateItem{}
  }
| INITIALLY IMMEDIATE
  {
    $$ = &ConstraintStateItem{}
  }
| INITIALLY DEFERRED
  {
    $$ = &ConstraintStateItem{}
  }
| VALIDATE
  {
    $$ = &ConstraintStateItem{Typ: TypValidate, ShouldValidate: true}
  }
| NOVALIDATE
  {
    $$ = &ConstraintStateItem{Typ: TypValidate}
  }
| RELY
  {
    $$ = &ConstraintStateItem{}
  }
| NORELY
  {
    $$ = &ConstraintStateItem{}
  }

constraint_detail:
  FOREIGN KEY opt_ident openb column_list closeb references
  {
    $$ = &ForeignKeyDetail{IndexName: $3, Source: $5, ReferenceInfo: $7}
  }
| PARTITION KEY openb sql_id closeb REFERENCES table_name openb sql_id closeb
  {
    $$ = &PartitionKeyDetail{Source: $4, ReferencedTable: $7, ReferencedColumn: $9}
  }
| check_constraint
  {
    $$ = $1
  }

foreign_primary_unique_definition:
  CONSTRAINT ID check_foreign_primary_unique_info
  {
    $$ = $3
    $$.Name = string($2)
  }
| check_foreign_primary_unique_info
  {
    $$ = $1
  }

check_foreign_primary_unique_info:
  foreign_primary_unique_detail
  {
    $$ = &ConstraintDefinition{Detail: $1}
  }
| check_constraint opt_constraint_state_items
  {
    $$ = &ConstraintDefinition{Detail: $1, State: mergeConstraintState($2)}
  }

check_constraint:
  CHECK openb condition closeb
  {
    $$ = &CheckCondition{CheckExpr: $3}
  }

foreign_primary_unique_detail:
  FOREIGN KEY opt_ident openb column_list closeb references
  {
    $$ = &ForeignKeyDetail{IndexName: $3, Source: $5, ReferenceInfo: $7}
  }
| PRIMARY KEY force_eof
  {
    $$ = &PrimaryKeyDetail{}
  }
| UNIQUE force_eof
  {
    $$ = &UniqueKeyDetail{}
  }

opt_references:
  {}
| references

references:
  REFERENCES table_name openb column_list closeb opt_on_update_delete
  {
    $$ = Reference{ReferencedTable: $2, ReferencedColumns: $4, Actions: $6}
  }

column_list:
  sql_id
  {
    $$ = Columns{$1}
  }
| column_list ',' sql_id
  {
    $$ = append($$, $3)
  }

opt_on_update_delete:
  {}
| fk_on_delete
  {
    $$.OnDelete = $1
  }
| fk_on_update
  {
    $$.OnUpdate = $1
  }
| fk_on_delete fk_on_update
  {
    $$.OnDelete = $1
    $$.OnUpdate = $2
  }
| fk_on_update fk_on_delete
  {
    $$.OnDelete = $2
    $$.OnUpdate = $1
  }

fk_on_delete:
  ON DELETE fk_reference_action
  {
    $$ = $3
  }

fk_on_update:
  ON UPDATE fk_reference_action
  {
    $$ = $3
  }

fk_reference_action:
  RESTRICT
  {
    $$ = Restrict
  }
| CASCADE
  {
    $$ = Cascade
  }
| NO ACTION
  {
    $$ = NoAction
  }
| SET DEFAULT
  {
    $$ = SetDefault
  }
| SET NULL
  {
    $$ = SetNull
  }

table_option_list:
  {
    $$ = TableOptions{}
  }
| table_option
  {
    $$ = $1
  }
| table_option_list ',' table_option
  {
    $$ = append($1, $3...)
  }

// rather than explicitly parsing the various keywords for table options,
// just accept any number of keywords, IDs, strings, numbers, and '='
table_option:
table_option_def
  {
    $$ = $1
  }
| table_option table_option_def
  {
    $$ = append($1, $2...)
  }

table_option_def:
  table_option_first_value table_opt_value
  {
    tableOption := TableOption{OptionName:$1, OptionValue:$2}
    // normalize engine, stripping off the quote chars
    if strings.EqualFold(tableOption.OptionName, "engine") {
        engine := tableOption.OptionValue
        if (engine[0]=='\'' && engine[len(engine)-1] == '\'') || (engine[0]=='"' && engine[len(engine)-1] == '"') {
            engine = engine[1 : len(engine)-1]
        }
        tableOption.OptionValue = engine
    }
    $$ = TableOptions{&tableOption}
  }
| table_option_first_value table_opt_value STORAGE storage_value
  {
    $$ = TableOptions{&TableOption{OptionName:$1, OptionValue:$2, OptionStorage: " storage " + $4}}
  }
| table_option_first_value '=' table_opt_value
  {
    tableOption := TableOption{OptionName:$1, OptionValue:$3}
    // normalize engine, stripping off the quote chars
    if strings.EqualFold(tableOption.OptionName, "engine") {
        engine := tableOption.OptionValue
        if (engine[0]=='\'' && engine[len(engine)-1] == '\'') || (engine[0]=='"' && engine[len(engine)-1] == '"') {
            engine = engine[1 : len(engine)-1]
        }
        tableOption.OptionValue = engine
    }
    $$ = TableOptions{&tableOption}
  }

table_option_first_value:
  AUTO_INCREMENT
  {
    $$ = string($1)
  }
| AVG_ROW_LENGTH
  {
    $$ = string($1)
  }
| DEFAULT CHARACTER SET
  {
      $$ = string($1) + " " + string($2) + " " +string($3)
  }
| CHARACTER SET
  {
      $$ = string($1)+ " " + string($2)
  }
| CHARSET
  {
      $$ = string($1)
  }
| DEFAULT CHARSET
  {
      $$ = string($1)+ " " + string($2)
  }
| COLLATE
   {
       $$ = string($1)
   }
| DEFAULT COLLATE
   {
       $$ = string($1)+ " " + string($2)
   }
| CHECKSUM
  {
      $$ = string($1)
  }
| COMMENT_KEYWORD
  {
      $$ = string($1)
  }
| COMPRESSION
  {
      $$ = string($1)
  }
| CONNECTION
  {
      $$ = string($1)
  }
| DATA DIRECTORY
  {
      $$ = string($1) + " " + string($2)
  }
| INDEX DIRECTORY
  {
      $$ = string($1) + " " + string($2)
  }
| DELAY_KEY_WRITE
  {
      $$ = string($1)
  }
| ENCRYPTION
  {
      $$ = string($1)
  }
| ENGINE
  {
      $$ = string($1)
  }
| INSERT_METHOD
  {
      $$ = string($1)
  }
| KEY_BLOCK_SIZE
  {
      $$ = string($1)
  }
| MAX_ROWS
  {
      $$ = string($1)
  }
| MIN_ROWS
  {
      $$ = string($1)
  }
| PACK_KEYS
  {
      $$ = string($1)
  }
| PASSWORD
  {
      $$ = string($1)
  }
| ROW_FORMAT
  {
      $$ = string($1)
  }
| STATS_AUTO_RECALC
  {
      $$ = string($1)
  }
| STATS_PERSISTENT
  {
      $$ = string($1)
  }
| STATS_SAMPLE_PAGES
  {
      $$ = string($1)
  }
| TABLESPACE
  {
      $$ = string($1)
  }
| UNION
  {
    $$ = string($1)
  }
| REMOTE_SEQ
  {
    $$ = string($1)
  }
| TYPE
  {
    $$ = string("engine")
  }

table_opt_value:
  reserved_sql_id
  {
    $$ = $1.String()
  }
| STRING
  {
    $$ = "'" + string($1) + "'"
  }
| INTEGRAL
  {
    $$ = string($1)
  }

storage_value:
  DISK
 {
  $$ = string($1)
 }
| MEMORY
 {
   $$ = string($1)
 }
| DEFAULT
  {
    $$ = string($1)
  }

key_part_list:
  '(' index_column_list ')'
  {
    $$ = $2
  }

opt_index_option_list:
  {
    $$ = nil
  }
| index_option_list
  {
    $$ = $1
  }

index_option_list:
  index_option
  {
    $$ = []*IndexOption{$1}
  }
| index_option_list index_option
  {
    $$ = append($$, $2)
  }

index_option:
 KEY_BLOCK_SIZE opt_equal opt_length
    {
      $$ = &IndexOption{KeyBlockSize: $3}
    }
  | USING BTREE
    {
      $$ = &IndexOption{Using: string($2)}
    }
  | USING HASH
    {
      $$ = &IndexOption{Using: string($2)}
    }
  | WITH PARSER ID
    {
      $$ = &IndexOption{PaserName: string($3)}
    }
  | COMMENT_KEYWORD STRING
    {
      $$ = &IndexOption{Comment: NewStrVal($2)}
    }

 opt_index_option:
   {
     $$ = nil
   }
 | index_option
   {
     $$ = $1
   }

 opt_algorithm_option:
   {
     $$ = ""
   }
 | ALGORITHM opt_equal algoritm_type
   {
     $$ = $3
   }

 opt_opt_lockion:
   {
     $$ = ""
   }
 | LOCK opt_equal lock_type
   {
     $$ = $3
   }

 algoritm_type:
   DEFAULT
   {
     $$ = string($1)
   }
 | INPLACE
   {
     $$ = string($1)
   }
 | COPY
   {
     $$ = string($1)
   }

 lock_type:
   DEFAULT
   {
     $$ = string($1)
   }
 | NONE
   {
     $$ = string($1)
   }
 | SHARED
   {
     $$ = string($1)
   }
 | EXCLUSIVE
   {
     $$ = string($1)
   }

alter_statement:
  ALTER DATABASE table_id ddl_force_eof
  {
    tmpSchemaSpec := &SchemaSpec{SchemaName: $3}
    $$ = &DDL{Action: AlterStr, SchemaSpec: tmpSchemaSpec, DDLType: SchemaDDL}
  }
| ALTER SCHEMA table_id ddl_force_eof
  {
    tmpSchemaSpec := &SchemaSpec{SchemaName: $3}
    $$ = &DDL{Action: AlterStr, SchemaSpec: tmpSchemaSpec, DDLType: SchemaDDL}
  }
| ALTER opt_ignore TABLE table_name non_add_drop_or_rename_operation force_eof
  {
    $$ = &DDL{Action: AlterStr, Table: $4, NewName: $4, DDLType: TableDDL}
  }
| ALTER opt_ignore TABLE table_name ADD add_partition_definition
  {
    tmpOptPartion := &OptCreatePartition{Definitions:[]*KunPartitionDefinition{$6}}
    $$ = &DDL{Action: AlterStr, AlterTyp: AlterAdd, Table: $4, NewName: $4, OptCreatePartition: tmpOptPartion, DDLType: TableDDL}
  }
| ALTER opt_ignore TABLE table_name ADD alter_object
  {
    $$ = &DDL{Action: AlterStr, AlterTyp: AlterAdd, Table: $4, NewName: $4, DDLType: TableDDL}
  }
| ALTER opt_ignore TABLE table_name DROP alter_object
  {
    $$ = &DDL{Action: AlterStr, AlterTyp: AlterDrop, Table: $4, NewName: $4, DDLType: TableDDL}
  }
| ALTER opt_ignore TABLE table_name ADD foreign_primary_unique_definition
  {
    $$ = &DDL{Action: AlterStr, AlterTyp: AlterAdd, AlterObject: NewAlterObject(ConstraintObj, $6.Name), Table: $4, NewName: $4, DDLType: TableDDL, ConstraintDef: $6}
  }

| ALTER opt_ignore TABLE table_name DROP PARTITION ID
  {
    tmpOptPartition := &OptPartition{Action: DropStr, Name: NewTableIdent(string($7))}
    $$ = &DDL{Action: AlterStr, Table: $4, NewName: $4, DDLType: TableDDL, OptPartition: tmpOptPartition}
  }
| ALTER opt_ignore TABLE table_name DROP CONSTRAINT ID
  {
    $$ = &DDL{Action: AlterStr, AlterTyp: AlterDrop, AlterObject: NewAlterObject(ConstraintObj, string($7)), Table: $4, NewName: $4, DDLType: TableDDL}
  }
| ALTER opt_ignore TABLE table_name TRUNCATE PARTITION ID
  {
    tmpOptPartition := &OptPartition{Action: TruncateStr, Name: NewTableIdent(string($7))}
    $$ = &DDL{Action: AlterStr, Table: $4, NewName: $4, DDLType: TableDDL, OptPartition: tmpOptPartition}
  }
| ALTER opt_ignore TABLE table_name MODIFY CONSTRAINT ID constraint_state_items
  {
    $$ = &DDL{Action: AlterStr, AlterTyp: AlterModify, AlterObject: NewAlterObject(ConstraintObj, string($7)), Table: $4, NewName: $4, DDLType: TableDDL, ConstraintState: mergeConstraintState($8)}
  }
| ALTER opt_ignore TABLE table_name MODIFY CONSTRAINT ID TO constraint_detail
  {
    $$ = &DDL{Action: AlterStr, AlterTyp: AlterModify, AlterObject: NewAlterObject(ConstraintObj, string($7)), Table: $4, NewName: $4, DDLType: TableDDL, ConstraintDef: NewConstraintDef(string($7), $9, nil)}
  }
| ALTER opt_ignore TABLE table_name RENAME CONSTRAINT ID TO ID
  {
    $$ = &DDL{Action: AlterStr, AlterTyp: AlterRename, AlterObject: NewAlterObject(ConstraintObj, string($7)), Table: TableName{Name: NewTableIdent(string($9))}, NewName: $4, DDLType: TableDDL}
  }
| ALTER opt_ignore TABLE table_name DROP opt_foreign_primary_unique force_eof
  {
   $$ = &DDL{Action: AlterStr, Table: $4, NewName: $4, DDLType: TableDDL}
  }
| ALTER opt_ignore TABLE table_name RENAME table_name
  {
    $$ = &DDL{Action: AlterRenameStr, Table: $4, NewName: $6, DDLType: TableDDL}
  }
| ALTER opt_ignore TABLE table_name RENAME TO table_name
  {
    $$ = &DDL{Action: AlterRenameStr, Table: $4, NewName: $7, DDLType: TableDDL}
  }
| ALTER opt_ignore TABLE table_name RENAME AS table_name
  {
    $$ = &DDL{Action: AlterRenameStr, Table: $4, NewName: $7, DDLType: TableDDL}
  }
| ALTER opt_ignore TABLE table_name RENAME PARTITION ID TO ID
  {
    tmpOptPartition := &OptPartition{Action: RenameStr, Name: NewTableIdent(string($7)), NewName: NewTableIdent(string($9))}
    $$ = &DDL{Action: AlterStr, Table: $4, NewName: $4, DDLType: TableDDL, OptPartition: tmpOptPartition}
  }
| ALTER opt_ignore TABLE table_name RENAME opt_index force_eof
  {
    // Rename an index can just be an alter
    $$ = &DDL{Action: AlterStr, Table: $4, NewName: $4, DDLType: TableDDL}
  }
| ALTER VIEW table_name AS select_statement opt_with_check
  {
    tmpViewSpec := &ViewSpec{ViewName:$3.ToViewName(), Select: $5, WithCheckOption: $6}
    $$ = &DDL{Action: AlterStr, Table: $3.ToViewName(), NewName: $3.ToViewName(), ViewSpec: tmpViewSpec, DDLType: ViewDDL}
  }
| ALTER opt_ignore TABLE table_name partition_operation
  {
    $$ = &DDL{Action: AlterStr, Table: $4, PartitionSpec: $5, DDLType: TableDDL}
  }
| ALTER TRIGGER table_name RENAME opt_to table_name
  {
     tmpTriggerSpec := &TriggerSpec{TriggerName:$3}
     $$ = &DDL{Action: AlterRenameStr, Table: $3, NewName: $6, TriggerSpec: tmpTriggerSpec, DDLType: TriggerDDL}
  }
| ALTER opt_ignore TABLE table_name change_operation
  {
    $$ = &DDL{Action: AlterStr, Table: $4, NewName: $4, ChangeSpec: $5, DDLType: TableDDL}
  }
| ALTER opt_ignore TABLE table_name modify_operation
  {
    $$ = &DDL{Action: AlterStr, Table: $4, NewName: $4, ModifySpec: $5, DDLType: TableDDL}
  }
| alter_function_stmt
  {
    $$ = $1
  }
| alter_procedure_stmt
  {
    $$ = $1
  }
| alter_package_stmt
  {
    $$ = $1
  }

opt_column:
  {
    $$ = nil
  }
| COLUMN
  {
    $$ = $1
  }

opt_order:
   {
     $$ = ""
   }
| FIRST
   {
     $$ = string($1)
   }
| AFTER ID
   {
     $$ = string($1) + " " + string($2)
   }

change_operation:
  CHANGE opt_column ID column_definition opt_order
  {
    $$ = &ChangeSpec{Action: ChangeStr, OldName: NewColIdent(string($3)), ColumnDefinition: $4, Order: $5}
  }

modify_operation:
  MODIFY opt_column column_definition opt_order
  {
     $$ = &ModifySpec{Action: ModifyStr , ColumnDefinition: $3, Order: $4}
  }

opt_interval:
  {
    $$ = nil
  }
| INTERVAL '(' value ')'
  {
    $$ = $3
  }

alter_object:
  openb force_eof
  {}
| alter_object_type force_eof
  {
    $$ = $1
  }

alter_object_type:
  COLUMN
| FULLTEXT
| ID
| INDEX
| KEY
| SPATIAL

partition_operation:
  REORGANIZE PARTITION sql_id INTO openb partition_definitions closeb
  {
    $$ = &PartitionSpec{Action: ReorganizeStr, Name: $3, Definitions: $6}
  }

partition_definitions:
  partition_definition
  {
    $$ = []*PartitionDefinition{$1}
  }
| partition_definitions ',' partition_definition
  {
    $$ = append($1, $3)
  }

partition_definition:
  PARTITION sql_id VALUES LESS THAN openb value_expression closeb
  {
    $$ = &PartitionDefinition{Name: $2, Limit: $7}
  }
| PARTITION sql_id VALUES LESS THAN openb MAXVALUE closeb
  {
    $$ = &PartitionDefinition{Name: $2, Maxvalue: true}
  }

range_partition_definitions:
  range_partition_definition
  {
    $$ = []*KunPartitionDefinition{$1}
  }
| range_partition_definitions ',' range_partition_definition
  {
    $$ = append($$, $3)
  }

range_partition_definition:
  PARTITION sql_id VALUES LESS THAN range_value_tuple
  {
    $$ = &KunPartitionDefinition{Name: $2, Rows: $6}
  }

list_partition_definitions:
  list_partition_definition
  {
    $$ = []*KunPartitionDefinition{$1}
  }
| list_partition_definitions ',' list_partition_definition
  {
    $$ = append($$, $3)
  }

list_partition_definition:
  PARTITION sql_id VALUES list_value_tuple
  {
    $$ = &KunPartitionDefinition{Name: $2, Rows: $4}
  }
| PARTITION sql_id VALUES IN list_value_tuple
  {
    $$ = &KunPartitionDefinition{Name: $2, Rows: $5}
  }

add_partition_definition:
  range_partition_definition
  {
    $$ = $1
  }
| PARTITION '(' range_partition_definition ')'
  {
    $$ = $3
  }
| list_partition_definition
  {
    $$ = $1
  }
| PARTITION '(' list_partition_definition ')'
  {
    $$ = $3
  }

rename_statement:
  RENAME TABLE table_name TO table_name
  {
    $$ = &DDL{Action: RenameStr, Table: $3, NewName: $5, DDLType: TableDDL}
  }

drop_statement:
  DROP DATABASE opt_exists table_id
  {
    var exists bool
    if $3 != 0 {
      exists = true
    }
    $$ = &DDL{Action: DropStr, IfExists: exists, SchemaSpec: &SchemaSpec{SchemaName:$4}, DDLType: SchemaDDL}
  }
| DROP SCHEMA opt_exists table_id
  {
    var exists bool
    if $3 != 0 {
      exists = true
    }
    $$ = &DDL{Action: DropStr, IfExists: exists, SchemaSpec: &SchemaSpec{SchemaName:$4}, DDLType: SchemaDDL}
  }
| DROP TABLE opt_exists table_name
  {
    var exists bool
    if $3 != 0 {
      exists = true
    }
    $$ = &DDL{Action: DropStr, Table: $4, NewName: $4, IfExists: exists, DDLType: TableDDL}
  }
| DROP INDEX ID ON table_name
  {
     tempIndexSpec := &IndexSpec{IdxDef: &IndexDefinition{Info: &IndexInfo{Type: "index", Name: NewTableIdent(string($3))}}}
     $$ = &DDL{Action: DropStr, IndexSpec: tempIndexSpec, Table: $5, NewName: $5, DDLType: IndexDDL}

  }
| DROP VIEW opt_comment opt_exists table_name ddl_force_eof
  {
    var exists bool
    if $4 != 0 {
      exists = true
    }
    tmpViewSpec := &ViewSpec{ViewName:$5.ToViewName()}
    $$ = &DDL{Action: DropStr, Table: $5.ToViewName(), Comments:Comments($3), IfExists: exists, DDLType: ViewDDL, ViewSpec: tmpViewSpec}
  }
| DROP SEQUENCE opt_exists table_name
  {
    var exists bool
    if $3 != 0 {
      exists = true
    }
    tmpSeqSpec := &SequenceSpec{FullName: $4, StartIndex:NewIntVal([]byte("0")), CacheSize:NewIntVal([]byte("1"))}
    $$ = &DDL{Action: DropStr, Table: $4, IfExists:exists, SequenceSpec: tmpSeqSpec, DDLType: SequenceDDL}
  }
| DROP TRIGGER opt_comment opt_exists table_name
  {
    var exists bool
    if $4 != 0 {
      exists = true
    }

    $$ = &DDL{Action: DropStr, Comments: Comments($3), IfExists:exists, Table: $5, TriggerSpec: &TriggerSpec{TriggerName:$5}, DDLType: TriggerDDL}
  }
| DROP SERVER opt_exists table_id
  {
    var exists bool
    if $3 != 0 {
      exists = true
    }
    $$ = &DDL{Action: DropStr, IfExists: exists, ServerSpec: &ServerSpec{ServerName:$4}, DDLType: ServerDDL}
  }
| DROP SYNONYM opt_exists table_name
  {
    var exists bool
    if $3 != 0 {
      exists = true
    }
    tmpSynonymSpec := &SynonymSpec{SynonymName: $4, TargetName: $4}
    $$ = &DDL{Action: DropStr, IfExists: exists, Table: $4, SynonymSpec: tmpSynonymSpec, DDLType: SynonymDDL}
  }
| drop_function_stmt
  {
    $$ = $1
  }
| drop_procedure_stmt
  {
    $$ = $1
  }
| drop_package_stmt
  {
    $$ = $1
  }

truncate_statement:
  TRUNCATE TABLE table_name
  {
    $$ = &DDL{Action: TruncateStr, Table: $3}
  }
| TRUNCATE table_name
  {
    $$ = &DDL{Action: TruncateStr, Table: $2}
  }


show_statement:
  SHOW opt_full TABLES db_from_spec show_condition
  {
    var full bool
    if $2 != 0 {
        full = true
    }
    $$ = &Show{Type: ShowTablesStr, FullShow: full, DBSpec: $4, Condition: $5}
  }
| SHOW TRIGGERS db_from_spec show_condition
  {
    $$ = &Show{Type: ShowTriggersStr, DBSpec: $3, Condition: $4}
  }
| SHOW opt_full COLUMNS tb_from_spec db_from_spec show_condition
  {
    var full bool
    if $2 != 0 {
        full = true
    }
    $$ = &Show{Type: ShowColumnsStr, FullShow: full, TbFromSpec: $4, DBSpec: $5, Condition: $6}
  }
| SHOW opt_full FIELDS tb_from_spec db_from_spec show_condition
  {
    var full bool
    if $2 != 0 {
        full = true
    }
    $$ = &Show{Type: ShowColumnsStr, FullShow: full, TbFromSpec: $4, DBSpec: $5, Condition: $6}
  }
| SHOW opt_global indexes_or_keys tb_from_spec db_from_spec opt_where_expression
  {
    var isGlobal bool = false
    if $2 != "" {
      isGlobal = true
    }
    $$ = &Show{Type: ShowIndexesStr, TbFromSpec: $4, DBSpec: $5, Condition: NewWhere(TypWhere, $6), IsGlobal: isGlobal}
  }
| SHOW TABLE STATUS db_from_spec show_condition
  {
    $$ = &Show{Type: ShowTableStatusStr, DBSpec: $4, Condition: $5}
  }
| SHOW PARTITIONS tb_from_spec
  {
    $$ = &Show{Type: ShowPartitionsStr, TbFromSpec: $3}
  }
| SHOW FUNCTION STATUS show_condition
  {
    $$ = &Show{Type: ShowFunctionStatusStr, Condition: $4}
  }
| SHOW PROCEDURE STATUS show_condition
  {
    $$ = &Show{Type: ShowProcedureStatusStr, Condition: $4}
  }
| SHOW PACKAGE STATUS show_condition
  {
    $$ = &Show{Type: ShowPackageStatusStr, Condition: $4}
  }
| SHOW PACKAGE BODY STATUS show_condition
  {
    $$ = &Show{Type: ShowPackageBodyStatusStr, Condition: $5}
  }
| SHOW CREATE SCHEMA table_id
  {
    $$ = &Show{Type: ShowCreateSchemaStr, DBSpec: NewDatabaseSpec($4)}
  }
| SHOW CREATE DATABASE table_id
  {
    $$ = &Show{Type: ShowCreateDatabaseStr, DBSpec: NewDatabaseSpec($4)}
  }
| SHOW CREATE TABLE table_name
  {
    $$ = &Show{Type: ShowCreateTableStr, TbFromSpec: NewTableFromSpec($4)}
  }
| SHOW CREATE VIEW table_name
  {
    $$ = &Show{Type: ShowCreateViewStr, TbFromSpec: NewTableFromSpec($4)}
  }
| SHOW CREATE PROCEDURE table_name
  {
    $$ = &Show{Type: ShowCreateProcedureStr, TbFromSpec: NewTableFromSpec($4)}
  }
| SHOW CREATE FUNCTION table_name
  {
    $$ = &Show{Type: ShowCreateFunctionStr, TbFromSpec: NewTableFromSpec($4)}
  }
| SHOW CREATE TRIGGER table_name
  {
    $$ = &Show{Type: ShowCreateTriggerStr, TbFromSpec: NewTableFromSpec($4)}
  }
| SHOW CREATE PACKAGE table_name
  {
    $$ = &Show{Type: ShowCreatePackageStr, TbFromSpec: NewTableFromSpec($4)}
  }
| SHOW CREATE PACKAGE BODY table_name
  {
    $$ = &Show{Type: ShowCreatePackageBodyStr, TbFromSpec: NewTableFromSpec($5)}
  }
| SHOW DATABASES force_eof
  {
    $$ = &Show{Type: ShowDatabasesStr}
  }
| SHOW SCHEMAS force_eof
  {
    $$ = &Show{Type: ShowSchemasStr}
  }
| SHOW GRANTS force_eof
  {
    $$ = &Show{Type: ShowGrantsStr}
  }
| SHOW GRANTS FOR grant_user
  {
    $$ = &Show{Type: ShowGrantsStr, Principal: $4}
  }
| SHOW GRANTS FOR ROLE grant_role
  {
    $$ = &Show{Type: ShowGrantsStr, Principal: $5}
  }
| SHOW KUNDB_KEYSPACES force_eof
  {
    $$ = &Show{Type: ShowKeyspacesStr}
  }
| SHOW KUNDB_SHARDS opt_table_name force_eof
  {
    $$ = &Show{Type: ShowShardsStr, TbFromSpec: NewTableFromSpec($3)}
  }
| SHOW SESSION SHARDS force_eof
  {
    $$ = &Show{Type: ShowSessionShardsStr}
  }
| SHOW VSCHEMA_TABLES force_eof
  {
    $$ = &Show{Type: ShowVSchemaTablesStr}
  }
| SHOW KUNDB_CHECKS FROM table_name
  {
    $$ = &Show{Type: ShowKunCheckStr, TbFromSpec: NewTableFromSpec($4)}
  }
| SHOW KUNDB_RANGE_INFO table_name
  {
    $$ = &Show{Type: ShowRangeInfoStr, TbFromSpec: NewTableFromSpec($3)}
  }
| SHOW KUNDB_VINDEXES tb_from_spec
  {
    $$ = &Show{Type: ShowVindexesStr, TbFromSpec: $3}
  }
| SHOW PRIVILEGES force_eof
  {
    $$ = &Show{Type: ShowPrivilegesStr}
  }
| SHOW variable_scope VARIABLES show_condition
  {
    $$ = &Show{Type: ShowVariablesStr, Condition: $4}
  }
| SHOW opt_full PROCESSLIST
  {
    $$ = &Show{Type: ShowProcesslistStr}
  }
| SHOW ENGINE force_eof
  {
    $$ = &Show{Type: ShowEngineStr}
  }
| SHOW opt_storage ENGINES force_eof
  {
    $$ = &Show{Type: ShowEnginesStr}
  }
| SHOW WARNINGS force_eof
  {
    $$ = &Show{Type: ShowWarningsStr}
  }
| SHOW ERRORS force_eof
  {
    $$ = &Show{Type: ShowErrorsStr}
  }
| SHOW CHARACTER SET force_eof
  {
    $$ = &Show{Type: ShowCharacterSetStr}
  }
| SHOW COLLATION force_eof
  {
    $$ = &Show{Type: ShowCollation}
  }
| SHOW unsupported_show_keyword force_eof
  {
    $$ = &Show{Type: ShowUnsupportedStr}
  }
| SHOW ID force_eof
  {
    $$ = &Show{Type: ShowUnsupportedStr}
  }

opt_global:
  {
    $$ = ""
  }
| GLOBAL
  {
    $$ = string($1)
  }

variable_scope:
  {}
| session_or_global
  { $$ = $1 }

opt_full:
  { $$ = 0 }
| FULL
  { $$ = 1 }

opt_storage:
  { $$ = 0 }
| STORAGE
  { $$ = 1 }

db_from_spec:
  {
    $$ = nil
  }
| FROM table_id
  {
    $$ = &DatabaseSpec{DBName: $2}
  }
| IN table_id
  {
    $$ = &DatabaseSpec{DBName: $2}
  }

tb_from_spec:
  FROM table_name
  {
    $$ = &TableFromSpec{TbName: $2}
  }
| IN table_name
  {
    $$ = &TableFromSpec{TbName: $2}
  }

show_condition:
  {
    $$ = nil
  }
| LIKE STRING
  {
    $$ = &ShowLikeExpr{LikeStr: string($2)}
  }
| WHERE expression
  {
    $$ = NewWhere(TypWhere, $2)
  }

use_statement:
  USE table_id
  {
    $$ = &Use{DBName: $2}
  }
| USE '@' ID
  {
    $$= &Use{DBName: NewTableIdent("@" + string($3))}
  }

call_statement:
  CALL proc_or_func_name openb opt_select_expression_list closeb
  {
    $$ = &Call{ProcName: $2}
  }

begin_statement:
  BEGIN
  {
    $$ = &Begin{}
  }
| START TRANSACTION
  {
    $$ = &Begin{}
  }

commit_statement:
  COMMIT
  {
    $$ = &Commit{}
  }

rollback_statement:
  ROLLBACK
  {
    $$ = &Rollback{}
  }

opt_explain_format:
  {
    $$ = ""

  }
| FORMAT '=' JSON
  {
    $$ = JSONStr
  }
| FORMAT '=' TREE
  {
    $$ = TreeStr
  }
| FORMAT '=' TRADITIONAL
  {
    $$ = TraditionalStr
  }

explain_synonyms:
  EXPLAIN
  {
    $$ = $1
  }
| DESCRIBE
  {
    $$ = $1
  }
| DESC
  {
    $$ = $1
  }

explainable_statement:
  select_statement
  {
    $$ = $1
  }
| update_statement
  {
    $$ = $1
  }
| insert_statement
  {
    $$ = $1
  }
| delete_statement
  {
    $$ = $1
  }

opt_wild:
  {
    $$ = ""
  }
| sql_id
  {
    $$ = ""
  }
| STRING
  {
    $$ = ""
  }

explain_statement:
  explain_synonyms table_name opt_wild
  {
    $$ = &OtherRead{OtherReadType: TableOrView, Object: $2}
  }
| explain_synonyms opt_explain_format explainable_statement
  {
    $$ = &Explain{Type: $2, Statement: $3}
  }

other_statement:
  REPAIR force_eof
  {
    $$ = &OtherAdmin{}
  }
| OPTIMIZE opt_analyze_optimize TABLE table_name
  {
    $$ = &OtherAdmin{Action: OptimizeStr, AdminOpt: $2, Table: $4}
  }
| ANALYZE opt_analyze_optimize TABLE table_name
  {
    $$ = &OtherAdmin{Action: AnalyzeStr, AdminOpt: $2, Table: $4}
  }

opt_analyze_optimize:
  {
    $$ = ""
  }
| NO_WRITE_TO_BINLOG
  {
    $$ = string($1)
  }
| LOCAL
  {
    $$ = string($1)
  }

opt_comment:
  {
    setAllowComments(yylex, true)
  }
  comment_list
  {
    $$ = $2
    setAllowComments(yylex, false)
  }

comment_list:
  {
    $$ = nil
  }
| comment_list COMMENT
  {
    $$ = append($1, $2)
  }

union_op:
  UNION
  {
    $$ = UnionStr
  }
| UNION ALL
  {
    $$ = UnionAllStr
  }
| UNION DISTINCT
  {
    $$ = UnionDistinctStr
  }

opt_cache:
{
  $$ = ""
}
| SQL_NO_CACHE
{
  $$ = SQLNoCacheStr
}
| SQL_CACHE
{
  $$ = SQLCacheStr
}

opt_distinct:
  {
    $$ = ""
  }
| DISTINCT
  {
    $$ = DistinctStr
  }

opt_straight_join:
  {
    $$ = ""
  }
| STRAIGHT_JOIN
  {
    $$ = StraightJoinHint
  }

opt_hint_list:
  {
    $$ = nil
  }
| hint_list
  {
    $$ = $1
  }

hint_list:
  hint
  {
    $$ = Hints{$1}
  }
| hint_list hint
  {
    $$ = append($$, $2)
  }

hint:
  BACKEND backend
  {
    $$ = &BackendHint{Backend: $2}
  }
| GLKJOIN '(' table_name_list ')'
  {
    $$ = &GlkjoinHint{TableNames: $3}
  }

backend:
  INCEPTOR
  {
    $$ = Inceptor
  }
| MFED
  {
    $$ = Mfed
  }

opt_select_expression_list:
  {
    $$ = nil
  }
| select_expression_list
  {
    $$ = $1
  }

select_expression_list:
  select_expression
  {
    $$ = SelectExprs{$1}
  }
| select_expression_list ',' select_expression
  {
    $$ = append($$, $3)
  }

select_expression:
  '*'
  {
    $$ = &StarExpr{}
  }
| expression opt_as_ci
  {
    var asOpt bool = false
    if $2 != nil {
       asOpt = true
    }
    $$ = &AliasedExpr{Expr: $1, AsOpt: asOpt, As: $2}
  }
| table_id '.' '*'
  {
    $$ = &StarExpr{TableName: TableName{Name: $1}}
  }
| table_id '.' reserved_table_id '.' '*'
  {
    $$ = &StarExpr{TableName: TableName{Qualifier: $1, Name: $3}}
  }

opt_as_ci:
  {
    $$ = nil
  }
| col_alias
  {
    $$ = $1
  }
| AS col_alias
  {
    $$ = $2
  }

col_alias:
  sql_id
  {
    $$ = NewStrVal([]byte($1.String()))
  }
| value
  {
    $$ = $1
  }

opt_from:
  {
    $$ = TableExprs{&AliasedTableExpr{Expr:TableName{Name: NewTableIdent("dual")}}}
  }
| FROM table_references
  {
    $$ = $2
  }

table_references:
  table_reference
  {
    $$ = TableExprs{$1}
  }
| table_references ','  table_reference
  {
    $$ = append($$, $3)
  }

table_reference:
  table_factor
| join_table

table_factor:
  aliased_table_name
  {
    $$ = $1
  }
| subquery opt_as table_id
  {
    $$ = &AliasedTableExpr{Expr:$1, As: $3}
  }
| subquery
  {
    $$ = &AliasedTableExpr{Expr:$1, As: NewTableIdent(nextAliasId(yylex))}
  }
| openb table_references closeb
  {
    $$ = &ParenTableExpr{Exprs: $2}
  }

aliased_table_name:
  table_name as_opt_id index_hint_list scan_mode_hint
  {
    $$ = &AliasedTableExpr{Expr:$1, As: $2, Hints: $3, ScanMode: $4}
  }
| table_name opt_partition_select as_opt_id index_hint_list scan_mode_hint
  {
    $$ = &AliasedTableExpr{Expr:$1, Partition: $2, As: $3, Hints: $4, ScanMode: $5}
  }

// There is a grammar conflict here:
// 1: INSERT INTO a SELECT * FROM b JOIN c ON b.i = c.i
// 2: INSERT INTO a SELECT * FROM b JOIN c ON DUPLICATE KEY UPDATE a.i = 1
// When yacc encounters the ON clause, it cannot determine which way to
// resolve. The %prec override below makes the parser choose the
// first construct, which automatically makes the second construct a
// syntax error. This is the same behavior as MySQL.
join_table:
  table_reference inner_join table_factor %prec JOIN
  {
    $$ = &JoinTableExpr{LeftExpr: $1, Join: $2, RightExpr: $3}
  }
| table_reference inner_join table_factor ON expression
  {
    $$ = &JoinTableExpr{LeftExpr: $1, Join: $2, RightExpr: $3, On: $5}
  }
| table_reference outer_join table_reference ON expression
  {
    $$ = &JoinTableExpr{LeftExpr: $1, Join: $2, RightExpr: $3, On: $5}
  }
| table_reference natural_join table_factor
  {
    $$ = &JoinTableExpr{LeftExpr: $1, Join: $2, RightExpr: $3}
  }

opt_as:
  { $$ = struct{}{} }
| AS
  { $$ = struct{}{} }

opt_partition_select:
  PARTITION '(' ID ')'
  {
    $$ = NewTableIdent(string($3))
  }

as_opt_id:
  {
    $$ = NewTableIdent("")
  }
| table_alias
  {
    $$ = $1
  }
| AS table_alias
  {
    $$ = $2
  }

table_alias:
  table_id
| STRING
  {
    $$ = NewTableIdent(string($1))
  }

inner_join:
  JOIN
  {
    $$ = JoinStr
  }
| INNER JOIN
  {
    $$ = JoinStr
  }
| CROSS JOIN
  {
    $$ = JoinStr
  }
| STRAIGHT_JOIN
  {
    $$ = StraightJoinStr
  }

outer_join:
  LEFT JOIN
  {
    $$ = LeftJoinStr
  }
| LEFT OUTER JOIN
  {
    $$ = LeftJoinStr
  }
| RIGHT JOIN
  {
    $$ = RightJoinStr
  }
| RIGHT OUTER JOIN
  {
    $$ = RightJoinStr
  }

natural_join:
 NATURAL JOIN
  {
    $$ = NaturalJoinStr
  }
| NATURAL outer_join
  {
    if $2 == LeftJoinStr {
      $$ = NaturalLeftJoinStr
    } else {
      $$ = NaturalRightJoinStr
    }
  }

into_table_name:
  INTO table_name
  {
    $$ = $2
  }
| table_name
  {
    $$ = $1
  }

opt_table_name:
  {
    $$ = TableName{Name: TableIdent{v: ""}}
  }
| table_name
  {
    $$ = $1
  }

table_name:
  table_id
  {
    $$ = TableName{Name: $1}
  }
| table_id '.' reserved_table_id
  {
    $$ = TableName{Qualifier: $1, Name: $3}
  }

delete_table_name:
table_id '.' '*'
  {
    $$ = TableName{Name: $1}
  }

proc_or_func_name:
  table_id
  {
    $$ = ProcOrFuncName{Name: $1}
  }
| table_id '.' reserved_table_id
  {
    $$ = ProcOrFuncName{Qualifier: $1, Name: $3}
  }
| table_id '.' table_id '.' reserved_table_id
  {
    $$ = ProcOrFuncName{Qualifier: $1, Package: $3, Name: $5}
  }

index_hint_list:
  {
    $$ = nil
  }
| USE INDEX openb index_list closeb
  {
    $$ = &IndexHints{Type: UseStr, Indexes: $4}
  }
| IGNORE INDEX openb index_list closeb
  {
    $$ = &IndexHints{Type: IgnoreStr, Indexes: $4}
  }
| FORCE INDEX openb index_list closeb
  {
    $$ = &IndexHints{Type: ForceStr, Indexes: $4}
  }

scan_mode_hint:
  {
    $$ = nil
  }
| FETCH MODE STRING
  {
    $$ = &ScanMode{Mode: NewStrVal($3)}
  }

index_list:
  sql_id
  {
    $$ = []ColumnDetail{NewColumnDetail($1, "")}
  }
| index_id
  {
   $$ = []ColumnDetail{$1}
  }
| index_list ',' sql_id
  {
    $$ = append($1, NewColumnDetail($3, ""))
  }
| index_list ',' index_id
  {
    $$ = append($1, $3)
  }

index_id:
  ID '(' opt_index_length ')'
  {
    $$ = NewColumnDetail(NewColIdent(string($1)), string($3))
  }

opt_index_length:
 INTEGRAL
  {
    $$ = $1
  }


part_list:
  sql_id
  {
    $$ = []ColIdent{$1}
  }
| part_list ',' sql_id
  {
    $$ = append($$, $3)
  }

opt_where_expression:
  {
    $$ = nil
  }
| WHERE expression
  {
    $$ = $2
  }

expression:
  condition
  {
    $$ = $1
  }
| expression ASSIGN expression
  {
    $$ = &AssignExpr{Left: $1, Right: $3}
  }
| expression AND expression
  {
    $$ = &AndExpr{Left: $1, Right: $3}
  }
| expression OR expression
  {
    $$ = &OrExpr{Left: $1, Right: $3}
  }
| NOT expression
  {
    $$ = &NotExpr{Expr: $2}
  }
| expression OP_CONCAT expression
  {
    $$ = &ConcatExpr{Left: $1, Right: $3}
  }
| expression IS is_suffix
  {
    $$ = &IsExpr{Operator: $3, Expr: $1}
  }
| value_expression
  {
    $$ = $1
  }
| DEFAULT opt_default
  {
    $$ = &Default{ColName: $2}
  }
| MAXVALUE
  {
    $$ = NewStrVal([]byte("maxvalue"))
  }

opt_default:
  /* empty */
  {
    $$ = ""
  }
| openb ID closeb
  {
    $$ = string($2)
  }

boolean_value:
  TRUE
  {
    $$ = BoolVal(true)
  }
| FALSE
  {
    $$ = BoolVal(false)
  }

condition:
  value_expression compare value_expression
  {
    $$ = &ComparisonExpr{Left: $1, Operator: $2, Right: $3}
  }
| value_expression IN col_tuple
  {
    $$ = &ComparisonExpr{Left: $1, Operator: InStr, Right: $3}
  }
| value_expression NOT IN col_tuple
  {
    $$ = &ComparisonExpr{Left: $1, Operator: NotInStr, Right: $4}
  }
| value_expression LIKE value_expression opt_like_escape
  {
    $$ = &ComparisonExpr{Left: $1, Operator: LikeStr, Right: $3, Escape: $4}
  }
| value_expression NOT LIKE value_expression opt_like_escape
  {
    $$ = &ComparisonExpr{Left: $1, Operator: NotLikeStr, Right: $4, Escape: $5}
  }
| value_expression REGEXP value_expression
  {
    $$ = &ComparisonExpr{Left: $1, Operator: RegexpStr, Right: $3}
  }
| value_expression NOT REGEXP value_expression
  {
    $$ = &ComparisonExpr{Left: $1, Operator: NotRegexpStr, Right: $4}
  }
| value_expression BETWEEN value_expression AND value_expression
  {
    $$ = &RangeCond{Left: $1, Operator: BetweenStr, From: $3, To: $5}
  }
| value_expression NOT BETWEEN value_expression AND value_expression
  {
    $$ = &RangeCond{Left: $1, Operator: NotBetweenStr, From: $4, To: $6}
  }
| EXISTS subquery
  {
    $$ = &ExistsExpr{Subquery: $2}
  }

is_suffix:
  NULL
  {
    $$ = IsNullStr
  }
| NOT NULL
  {
    $$ = IsNotNullStr
  }
| TRUE
  {
    $$ = IsTrueStr
  }
| NOT TRUE
  {
    $$ = IsNotTrueStr
  }
| FALSE
  {
    $$ = IsFalseStr
  }
| NOT FALSE
  {
    $$ = IsNotFalseStr
  }

compare:
  '='
  {
    $$ = EqualStr
  }
| '<'
  {
    $$ = LessThanStr
  }
| '>'
  {
    $$ = GreaterThanStr
  }
| LE
  {
    $$ = LessEqualStr
  }
| GE
  {
    $$ = GreaterEqualStr
  }
| NE
  {
    $$ = NotEqualStr
  }
| NULL_SAFE_EQUAL
  {
    $$ = NullSafeEqualStr
  }

opt_like_escape:
  {
    $$ = nil
  }
| ESCAPE value_expression
  {
    $$ = $2
  }

col_tuple:
  row_tuple
  {
    $$ = $1
  }
| subquery
  {
    $$ = $1
  }
| LIST_ARG
  {
    $$ = ListArg($1)
  }

subquery:
  openb select_statement closeb
  {
    $$ = &Subquery{$2}
  }

expression_list:
  expression
  {
    $$ = Exprs{$1}
  }
| expression_list ',' expression
  {
    $$ = append($1, $3)
  }

range_value_tuple:
  openb range_expression_list closeb
  {
    $$ = ValTuple($2)
  }

range_expression_list:
  range_expression
  {
    $$ = Exprs{$1}
  }
| range_expression_list ',' range_expression
  {
    $$ = append($1, $3)
  }

list_value_tuple:
  openb list_expression_list closeb
  {
    $$ = ValTuple($2)
  }

list_expression_list:
  list_expression
  {
    $$ = Exprs{$1}
  }
| list_expression_list ',' list_expression
  {
    $$ = append($1, $3)
  }

range_expression:
  value_expression
    {
      $$ = $1
    }
  | MAXVALUE
    {
      $$ = NewStrVal([]byte("maxvalue"))
    }

list_expression:
  value_expression
    {
      $$ = $1
    }
  | DEFAULT
    {
      $$ = NewStrVal([]byte("default"))
    }

value_expression:
  value
  {
    $$ = $1
  }
| boolean_value
  {
    $$ = $1
  }
| column_name
  {
    $$ = $1
  }
| rownum
  {
    $$ = $1
  }
| tuple_expression
  {
    $$ = $1
  }
| subquery
  {
    $$ = $1
  }
| value_expression '&' value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: BitAndStr, Right: $3}
  }
| value_expression '|' value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: BitOrStr, Right: $3}
  }
| value_expression '^' value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: BitXorStr, Right: $3}
  }
| value_expression '+' value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: PlusStr, Right: $3}
  }
| value_expression '-' value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: MinusStr, Right: $3}
  }
| value_expression '*' value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: MultStr, Right: $3}
  }
| value_expression '/' value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: DivStr, Right: $3}
  }
| value_expression DIV value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: IntDivStr, Right: $3}
  }
| value_expression '%' value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: ModStr, Right: $3}
  }
| value_expression MOD value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: ModStr, Right: $3}
  }
| value_expression SHIFT_LEFT value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: ShiftLeftStr, Right: $3}
  }
| value_expression SHIFT_RIGHT value_expression
  {
    $$ = &BinaryExpr{Left: $1, Operator: ShiftRightStr, Right: $3}
  }
| column_name JSON_EXTRACT_OP value
  {
    $$ = &BinaryExpr{Left: $1, Operator: JSONExtractOp, Right: $3}
  }
| column_name JSON_UNQUOTE_EXTRACT_OP value
  {
    $$ = &BinaryExpr{Left: $1, Operator: JSONUnquoteExtractOp, Right: $3}
  }
| value_expression COLLATE charset
  {
    $$ = &CollateExpr{Expr: $1, Charset: $3}
  }
| BINARY value_expression %prec UNARY
  {
    $$ = &UnaryExpr{Operator: BinaryStr, Expr: $2}
  }
| UNDERSCORECS value_expression %prec UNARY
  {
    $$ = &UnaryExpr{Operator: string($1), Expr: $2}
  }
| '+'  value_expression %prec UNARY
  {
    if num, ok := $2.(*SQLVal); ok && num.Type == IntVal {
      $$ = num
    } else {
      $$ = &UnaryExpr{Operator: UPlusStr, Expr: $2}
    }
  }
| '-'  value_expression %prec UNARY
  {
    if num, ok := $2.(*SQLVal); ok && num.Type == IntVal {
      // Handle double negative
      if num.Val[0] == '-' {
        num.Val = num.Val[1:]
        $$ = num
      } else {
        $$ = NewIntVal(append([]byte("-"), num.Val...))
      }
    } else {
      $$ = &UnaryExpr{Operator: UMinusStr, Expr: $2}
    }
  }
| '~'  value_expression
  {
    $$ = &UnaryExpr{Operator: TildaStr, Expr: $2}
  }
| '!' value_expression %prec UNARY
  {
    $$ = &UnaryExpr{Operator: BangStr, Expr: $2}
  }
| INTERVAL value_expression sql_id
  {
    // This rule prevents the usage of INTERVAL
    // as a function. If support is needed for that,
    // we'll need to revisit this. The solution
    // will be non-trivial because of grammar conflicts.
    $$ = &IntervalExpr{Expr: $2, Unit: $3}
  }
| function_call_generic
| function_call_keyword
| function_call_nonkeyword
| function_call_conflict

/*
  Regular function calls without special token or syntax, guaranteed to not
  introduce side effects due to being a simple identifier
*/
function_call_generic:
  sql_id openb opt_select_expression_list closeb
  {
    if $1.Lowered() == "nextval" {
      setHasSequenceUpdateNode(yylex, true)
    }
    $$ = &FuncExpr{Name: $1, Exprs: $3}
  }
| sql_id openb DISTINCT select_expression_list closeb
  {
    $$ = &FuncExpr{Name: $1, Distinct: true, Exprs: $4}
  }
| table_id '.' reserved_sql_id openb opt_select_expression_list closeb
  {
    $$ = &FuncExpr{Qualifier: $1, Name: $3, Exprs: $5}
  }

/*
  Function calls using reserved keywords, with dedicated grammar rules
  as a result
*/
function_call_keyword:
  LEFT openb select_expression_list closeb
  {
    $$ = &FuncExpr{Name: NewColIdent("left"), Exprs: $3}
  }
| RIGHT openb select_expression_list closeb
  {
    $$ = &FuncExpr{Name: NewColIdent("right"), Exprs: $3}
  }
| INSERT openb select_expression_list closeb
   {
     $$ = &FuncExpr{Name: NewColIdent("insert"), Exprs: $3}
   }
| CONVERT openb expression ',' convert_type closeb
  {
    $$ = &ConvertExpr{Expr: $3, Type: $5}
  }
| CAST openb expression AS convert_type closeb
  {
    $$ = &ConvertExpr{Expr: $3, Type: $5}
  }
| CONVERT openb expression USING charset closeb
  {
    $$ = &ConvertUsingExpr{Expr: $3, Type: $5}
  }
| MATCH openb select_expression_list closeb AGAINST openb value_expression match_option closeb
  {
  $$ = &MatchExpr{Columns: $3, Expr: $7, Option: $8}
  }
| GROUP_CONCAT openb opt_distinct select_expression_list opt_order_by opt_separator closeb
  {
    $$ = &GroupConcatExpr{Distinct: $3, Exprs: $4, OrderBy: $5, Separator: $6}
  }
| CASE opt_expression when_expression_list opt_else_expression END
  {
    $$ = &CaseExpr{Expr: $2, Whens: $3, Else: $4}
  }
| VALUES openb sql_id closeb
  {
    $$ = &ValuesFuncExpr{Name: $3}
  }
| CONCAT openb expression_list closeb
  {
    $$ = &ConcatExpr{Exprs: $3}
  }

/*
  Function calls using non reserved keywords but with special syntax forms.
  Dedicated grammar rules are needed because of the special syntax
*/
function_call_nonkeyword:
  CURRENT_TIMESTAMP opt_func_datetime_precision
  {
    $$ = &FuncExpr{Name:NewColIdent("current_timestamp"), Exprs:$2}
  }
| UTC_TIMESTAMP opt_func_datetime_precision
  {
    $$ = &FuncExpr{Name:NewColIdent("utc_timestamp"), Exprs:$2}
  }
| UTC_TIME opt_func_datetime_precision
  {
    $$ = &FuncExpr{Name:NewColIdent("utc_time"), Exprs:$2}
  }
| UTC_DATE opt_func_datetime_precision
  {
    $$ = &FuncExpr{Name:NewColIdent("utc_date"), Exprs:$2}
  }
  // now
| LOCALTIME opt_func_datetime_precision
  {
    $$ = &FuncExpr{Name:NewColIdent("localtime"), Exprs:$2}
  }
  // now
| LOCALTIMESTAMP opt_func_datetime_precision
  {
    $$ = &FuncExpr{Name:NewColIdent("localtimestamp"), Exprs:$2}
  }
  // curdate
| CURRENT_DATE opt_func_datetime_precision
  {
    $$ = &FuncExpr{Name:NewColIdent("current_date"), Exprs:$2}
  }
  // curtime
| CURRENT_TIME opt_func_datetime_precision
  {
    $$ = &FuncExpr{Name:NewColIdent("current_time"), Exprs:$2}
  }

opt_func_datetime_precision:
  /* empty */
  {
    $$ = nil
  }
  /* support time func with precision like current_timestamp(6) */
| openb opt_select_expression_list closeb
  {
    err := checkDateTimePrecisionOption($2)
    if err == nil {
      $$ = $2
    } else {
      yylex.Error(err.Error())
      return 1
    }
  }

/*
  Function calls using non reserved keywords with *normal* syntax forms. Because
  the names are non-reserved, they need a dedicated rule so as not to conflict
*/
function_call_conflict:
  IF openb select_expression_list closeb
  {
    $$ = &FuncExpr{Name: NewColIdent("if"), Exprs: $3}
  }
| DATABASE openb opt_select_expression_list closeb
  {
    $$ = &FuncExpr{Name: NewColIdent("database"), Exprs: $3}
  }
| MOD openb select_expression_list closeb
  {
    $$ = &FuncExpr{Name: NewColIdent("mod"), Exprs: $3}
  }
| REPLACE openb select_expression_list closeb
  {
    $$ = &FuncExpr{Name: NewColIdent("replace"), Exprs: $3}
  }

match_option:
/*empty*/
  {
    $$ = ""
  }
| IN BOOLEAN MODE
  {
    $$ = BooleanModeStr
  }
| IN NATURAL LANGUAGE MODE
 {
    $$ = NaturalLanguageModeStr
 }
| IN NATURAL LANGUAGE MODE WITH QUERY EXPANSION
 {
    $$ = NaturalLanguageModeWithQueryExpansionStr
 }
| WITH QUERY EXPANSION
 {
    $$ = QueryExpansionStr
 }

charset:
  ID
{
    $$ = string($1)
}
| STRING
{
    $$ = string($1)
}

convert_type:
  BINARY opt_length
  {
    $$ = &ConvertType{Type: string($1), Length: $2}
  }
| CHAR opt_length opt_charset
  {
    $$ = &ConvertType{Type: string($1), Length: $2, Charset: $3, Operator: CharacterSetStr}
  }
| CHAR opt_length ID
  {
    $$ = &ConvertType{Type: string($1), Length: $2, Charset: string($3)}
  }
| DATE
  {
    $$ = &ConvertType{Type: string($1)}
  }
| DATETIME opt_length
  {
    $$ = &ConvertType{Type: string($1), Length: $2}
  }
| DECIMAL opt_decimal_length
  {
    $$ = &ConvertType{Type: string($1)}
    $$.Length = $2.Length
    $$.Scale = $2.Scale
  }
| JSON
  {
    $$ = &ConvertType{Type: string($1)}
  }
| NCHAR opt_length
  {
    $$ = &ConvertType{Type: string($1), Length: $2}
  }
| SIGNED
  {
    $$ = &ConvertType{Type: string($1)}
  }
| SIGNED INTEGER
  {
    $$ = &ConvertType{Type: string($1)}
  }
| TIME opt_length
  {
    $$ = &ConvertType{Type: string($1), Length: $2}
  }
| UNSIGNED
  {
    $$ = &ConvertType{Type: string($1)}
  }
| UNSIGNED INTEGER
  {
    $$ = &ConvertType{Type: string($1)}
  }

opt_expression:
  {
    $$ = nil
  }
| expression
  {
    $$ = $1
  }

opt_separator:
  {
    $$ = string("")
  }
| SEPARATOR STRING
  {
    $$ = " separator '"+string($2)+"'"
  }

when_expression_list:
  when_expression
  {
    $$ = []*When{$1}
  }
| when_expression_list when_expression
  {
    $$ = append($1, $2)
  }

when_expression:
  WHEN expression THEN expression
  {
    $$ = &When{Cond: $2, Val: $4}
  }

opt_else_expression:
  {
    $$ = nil
  }
| ELSE expression
  {
    $$ = $2
  }

rownum:
  ROWNUM
  {
    setHasRownum(yylex, true)
    $$ = &RownumExpr{}
  }

column_name:
  sql_id
  {
    $$ = &ColName{Name: $1}
  }
| table_id '.' reserved_sql_id
  {
    $$ = &ColName{Qualifier: TableName{Name: $1}, Name: $3}
  }
| table_id '.' reserved_table_id '.' reserved_sql_id
  {
    $$ = &ColName{Qualifier: TableName{Qualifier: $1, Name: $3}, Name: $5}
  }

value:
  STRING
  {
    $$ = NewStrVal($1)
  }
| HEX
  {
    $$ = NewHexVal($1)
  }
| BIT_LITERAL
  {
    $$ = NewBitVal($1)
  }
| INTEGRAL
  {
    $$ = NewIntVal($1)
  }
| FLOAT
  {
    $$ = NewFloatVal($1)
  }
| HEXNUM
  {
    $$ = NewHexNum($1)
  }
| VALUE_ARG
  {
    $$ = NewValArg($1)
  }
| NULL
  {
    $$ = &NullVal{}
  }

num_val:
  sql_id
  {
    // TODO(sougou): Deprecate this construct.
    if $1.Lowered() != "value" {
      yylex.Error("expecting value after next")
      return 1
    }
    $$ = NewIntVal([]byte("1"))
  }

opt_group_by:
  {
    $$ = nil
  }
| GROUP BY expression_list
  {
    $$ = $3
  }

opt_having:
  {
    $$ = nil
  }
| HAVING expression
  {
    $$ = $2
  }

opt_order_by:
  {
    $$ = nil
  }
| ORDER BY order_list
  {
    $$ = $3
  }

order_list:
  order
  {
    $$ = OrderBy{$1}
  }
| order_list ',' order
  {
    $$ = append($1, $3)
  }

order:
  expression opt_asc_desc
  {
    $$ = &Order{Expr: $1, Direction: $2}
  }

opt_asc_desc:
  {
    $$ = AscScr
  }
| ASC
  {
    $$ = AscScr
  }
| DESC
  {
    $$ = DescScr
  }

opt_limit:
  {
    $$ = nil
  }
| LIMIT expression
  {
    setHasLimit(yylex, true)
    $$ = &Limit{Rowcount: $2}
  }
| LIMIT expression ',' expression
  {
    setHasLimit(yylex, true)
    $$ = &Limit{Offset: $2, Rowcount: $4}
  }
| LIMIT expression OFFSET expression
  {
    setHasLimit(yylex, true)
    $$ = &Limit{Offset: $4, Rowcount: $2}
  }

opt_definer:
  {
    $$ = ""
  }
| DEFINER '=' user_or_role
  {
    $$ = "DEFINER=" + $3
  }
user_or_role:
  ID '@' ID
  {
    $$ = string($1) + "@" + string($3)
  }

view_algorithm:
  ALGORITHM '=' UNDEFINED
  {
    $$= "ALGORITHM=UNDEFINED"
  }
| ALGORITHM '=' MERGE
  {
    $$= "ALGORITHM=MERGE"
  }
| ALGORITHM '=' TEMPTABLE
  {
    $$= "ALGORITHM=TEMPTABLE"
  }

opt_view_suid:
  {
    $$ = ""
  }
| view_suid
  {
    $$ = $1
  }

view_suid:
  SQL SECURITY DEFINER
  {
    $$ = "SQL SECURITY DEFINER"
  }
| SQL SECURITY INVOKER
  {
    $$ = "SQL SECURITY INVOKER"
  }
opt_lock:
  {
    $$ = ""
  }
| FOR UPDATE
  {
    $$ = ForUpdateStr
  }
| LOCK IN SHARE MODE
  {
    $$ = ShareModeStr
  }

// insert_data expands all combinations into a single rule.
// This avoids a shift/reduce conflict while encountering the
// following two possible constructs:
// insert into t1(a, b) (select * from t2)
// insert into t1(select * from t2)
// Because the rules are together, the parser can keep shifting
// the tokens until it disambiguates a as sql_id and select as keyword.
insert_data:
  VALUES tuple_list
  {
    $$ = &Insert{Rows: $2}
  }
| select_statement
  {
    $$ = &Insert{Rows: $1}
  }
| openb select_statement closeb
  {
    // Drop the redundant parenthesis.
    $$ = &Insert{Rows: $2}
  }
| openb ins_column_list closeb VALUES tuple_list
  {
    $$ = &Insert{Columns: $2, Rows: $5}
  }
| openb ins_column_list closeb select_statement
  {
    $$ = &Insert{Columns: $2, Rows: $4}
  }
| openb ins_column_list closeb openb select_statement closeb
  {
    // Drop the redundant parenthesis.
    $$ = &Insert{Columns: $2, Rows: $5}
  }

ins_column_list:
  sql_id
  {
    $$ = Columns{$1}
  }
| sql_id '.' sql_id
  {
    $$ = Columns{$3}
  }
| ins_column_list ',' sql_id
  {
    $$ = append($$, $3)
  }
| ins_column_list ',' sql_id '.' sql_id
  {
    $$ = append($$, $5)
  }

opt_on_dup:
  {
    $$ = nil
  }
| ON DUPLICATE KEY UPDATE update_list
  {
    $$ = $5
  }

tuple_list:
  tuple_or_empty
  {
    $$ = Values{$1}
  }
| tuple_list ',' tuple_or_empty
  {
    $$ = append($1, $3)
  }

tuple_or_empty:
  row_tuple
  {
    $$ = $1
  }
| openb closeb
  {
    $$ = ValTuple{}
  }

row_tuple:
  openb expression_list closeb
  {
    $$ = ValTuple($2)
  }

tuple_expression:
  row_tuple
  {
    if len($1) == 1 {
      $$ = &ParenExpr{$1[0]}
    } else {
      $$ = $1
    }
  }

update_list:
  update_expression
  {
    $$ = UpdateExprs{$1}
  }
| update_list ',' update_expression
  {
    $$ = append($1, $3)
  }

update_expression:
  column_name '=' expression
  {
    $$ = &UpdateExpr{Name: $1, Expr: $3}
  }

set_list:
  set_expression
  {
    $$ = SetExprs{$1}
  }
| set_list ',' set_expression
  {
    $$ = append($1, $3)
  }

set_expression:
reserved_sql_id '=' ON
  {
    $$ = &SetExpr{Name: $1, Expr: NewStrVal([]byte("on"))}
  }
| reserved_sql_id '=' OFF
  {
    $$ = &SetExpr{Name: $1, Expr: NewStrVal([]byte("off"))}
  }
| reserved_sql_id '=' expression
  {
    $$ = &SetExpr{Name: $1, Expr: $3}
  }
| charset_or_character_set charset_value opt_collate
  {
    $$ = &SetExpr{Name: NewColIdent(string($1)), Expr: $2}
  }

charset_or_character_set:
  CHARSET
| CHARACTER SET
  {
    $$ = []byte("charset")
  }
| NAMES

charset_value:
  sql_id
  {
    $$ = NewStrVal([]byte($1.String()))
  }
| STRING
  {
    $$ = NewStrVal($1)
  }
| DEFAULT
  {
    $$ = &Default{}
  }

opt_exists:
  { $$ = 0 }
| IF EXISTS
  { $$ = 1 }

opt_not_exists:
  { $$ = 0 }
| IF NOT EXISTS
  { $$ = 1 }

opt_temporary:
  { $$ = 0 }
| TEMPORARY
  { $$ = 1 }

opt_ignore:
  { $$ = "" }
| IGNORE
  { $$ = IgnoreStr }

non_add_drop_or_rename_operation:
  ALGORITHM
  { $$ = struct{}{} }
| ALTER
  { $$ = struct{}{} }
| AUTO_INCREMENT
  { $$ = struct{}{} }
| CHARACTER
  { $$ = struct{}{} }
| COMMENT_KEYWORD
  { $$ = struct{}{} }
| DEFAULT
  { $$ = struct{}{} }
| LOCK
  { $$ = struct{}{} }
| ORDER
  { $$ = struct{}{} }
| CONVERT
  { $$ = struct{}{} }
| PARTITION
  { $$ = struct{}{} }
| UNUSED
  { $$ = struct{}{} }
| ENABLE
  { $$ = struct{}{} }
| DISABLE
  { $$ = struct{}{} }
| ID
  { $$ = struct{}{} }

opt_to:
  { $$ = struct{}{} }
| TO
  { $$ = struct{}{} }
| AS
  { $$ = struct{}{} }

opt_index:
  INDEX
  { $$ = struct{}{} }
| KEY
  { $$ = struct{}{} }

opt_constraint_name:
  {
    $$ = ""
  }
| CONSTRAINT ID
  {
    $$ = string($2)
  }

opt_constraint:
  {
    $$ = string("")
  }
| UNIQUE
  {
    $$ = string($1)
  }
| FULLTEXT
  {
    $$ = string($1)
  }
| SPATIAL
  {
    $$ = string($1)
  }

opt_foreign_primary_unique:
  UNIQUE
  {
    $$ = string($1)
  }
| FOREIGN
  {
    $$ = string($1)
  }
| PRIMARY
  {
    $$ = string($1)
  }
opt_index_type:
  {
    $$ = ""
  }
| USING BTREE
  {
    $$ = string($2)
  }
| USING HASH
  {
    $$ = string($2)
  }

opt_partition_using:
 USING sql_id
  { $$ = $2}

sql_id:
  ID
  {
    $$ = NewColIdent(string($1))
  }
| non_reserved_keyword
  {
    $$ = NewColIdent(string($1))
  }

reserved_sql_id:
  sql_id
| reserved_keyword
  {
    $$ = NewColIdent(string($1))
  }

table_id:
  ID
  {
    $$ = NewTableIdent(string($1))
  }
| non_reserved_keyword
  {
    $$ = NewTableIdent(string($1))
  }

reserved_table_id:
  table_id
| reserved_keyword
  {
    $$ = NewTableIdent(string($1))
  }

opt_ident:
  {}
| ident
  {
    $$ = $1
  }

ident:
  ID
  {
    $$ = string($1)
  }
| non_reserved_keyword
  {
    $$ = string($1)
  }

/*
  These are not all necessarily reserved in MySQL, but some are.

  These are more importantly reserved because they may conflict with our grammar.
  If you want to move one that is not reserved in MySQL (i.e. ESCAPE) to the
  non_reserved_keyword, you'll need to deal with any conflicts.

  Sorted alphabetically
*/
reserved_keyword:
  ADD
| AFTER
| ALGORITHM
| ALWAYS
| AND
| AS
| ASC
| AUTO_INCREMENT
| BACKEND
| BEFORE
| BETWEEN
| BINARY
| BY
| CALL
| CASE
| COLLATE
| COLUMNS
| CONSTRAINT
| CONVERT
| CREATE
| CROSS
| CURRENT_DATE
| CURRENT_TIME
| CURRENT_TIMESTAMP
| DATABASE
| SCHEMA
| DATABASES
| SCHEMAS
| DECLARE
| DEFAULT
| DELETE
| DESC
| DESCRIBE
| DISTINCT
| DIV
| DROP
| EACH
| ELSE
| END
| ESCAPE
| EXISTS
| EXPLAIN
| FALSE
| FIELDS
| FOR
| FORCE
| FROM
| FULL
| FUNCTION
| GENERATED
| GLKJOIN
| GROUP
| HAVING
| IF
| IGNORE
| IN
| INDEX
| INDEXES
| INNER
| INSERT
| INTERVAL
| INTO
| INVOKER
| IS
| JOIN
| KEY
| KEYS
| LEFT
| LIKE
| LIMIT
| LOCALTIME
| LOCALTIMESTAMP
| LOCK
| MATCH
| MAXVALUE
| MERGE
| MOD
| NATURAL
| NEXT // next should be doable as non-reserved, but is not due to the special `select next num_val` query that vitess supports
| NOT
| NULL
| OFF
| ON
| OR
| ORDER
| OUTER
| PROCEDURE
| PACKAGE
| PACKAGES
| PARTITIONS
| REGEXP
| RENAME
| REPLACE
| ROLE
| ROWNUM
| RIGHT
| SELECT
| SEPARATOR
| SESSION
| SET
| SHOW
| STORED
| STRAIGHT_JOIN
| SYNONYM
| TABLE
| TABLES
| TEMPTABLE
| THEN
| TO
| TRIGGER
| TRIGGERS
| TRUE
| UNDEFINED
| UNION
| UNIQUE
| UPDATE
| USE
| USING
| UTC_DATE
| UTC_TIME
| UTC_TIMESTAMP
| VALUES
| VARIABLES
| VIEW
| WHEN
| WHERE
| WITH


/*
  These are non-reserved Vitess, because they don't cause conflicts in the grammar.
  Some of them may be reserved in MySQL. The good news is we backtick quote them
  when we rewrite the query, so no issue should arise.

  Sorted alphabetically
*/
non_reserved_keyword:
  ACTION
| ADMIN
| AGAINST
| ALL
| AVG_ROW_LENGTH
| BEGIN
| BIGINT
| BINARY_MD5
| BIT
| BLOB
| BOOL
| BOOLEAN
| BTREE
| CACHE
| CHAR
| CHARACTER
| CHARSET
| CHECKSUM
| CLOB
| COLLATION
| COMMENT_KEYWORD
| COMMIT
| COMMITTED
| COMPRESSION
| CONNECTION
| COPY
| DATA
| DATE
| DATETIME
| DECIMAL
| DEFINER
| DELAY_KEY_WRITE
| DIRECTORY
| DISK
| DOUBLE
| DUPLICATE
| ENCRYPTION
| ENGINE
| ENGINES
| ENUM
| EXCLUSIVE
| EXECUTE
| EXPANSION
| FLOAT_TYPE
| FOREIGN
| FORMAT
| FULLTEXT
| GEOMETRY
| GEOMETRYCOLLECTION
| GLOBAL
| GRANT
| GRANTS
| HASH
| INCEPTOR
| INPLACE
| INSERT_METHOD
| INT
| INTEGER
| ISOLATION
| JSON
| KEY_BLOCK_SIZE
| LANGUAGE
| LAST_INSERT_ID
| LESS
| LEVEL
| LINESTRING
| LIST
| LOCAL
| LOCATE
| LONGBLOB
| LONGTEXT
| MAX_ROWS
| MEDIUMBLOB
| MEDIUMINT
| MEDIUMTEXT
| MEMORY
| MFED
| MIN_ROWS
| MODE
| MULTILINESTRING
| MULTIPOINT
| MULTIPOLYGON
| NAME
| NAMES
| NCHAR
| NO
| NO_WRITE_TO_BINLOG
| NONE
| NUMERIC
| NUMERIC_STATIC_MAP
| OFFSET
| ONLY
| OPTIMIZE
| OPTION
| PACK_KEYS
| PARSER
| PARTITION
| PASSWORD
| POINT
| POLYGON
| PRIMARY
| PRIVILEGES
| QUERY
| RANGE
| READ
| REAL
| REFERENCE
| REFERENCES
| REORGANIZE
| REPAIR
| REPEATABLE
| REPLICATION
| REVERSE_BITS
| REVOKE
| ROLLBACK
| ROUTINE
| ROW
| ROW_FORMAT
| SECURITY
| SEQUENCE
| SERIALIZABLE
| SHARDS
| SHARE
| SHARED
| SIGNED
| SMALLINT
| SPATIAL
| SQL
| STATS_AUTO_RECALC
| STATS_PERSISTENT
| STATS_SAMPLE_PAGES
| STATUS
| STORAGE
| TABLESPACE
| TEMPORARY
| TEXT
| THAN
| TIME
| TIMESTAMP
| TINYBLOB
| TINYINT
| TINYTEXT
| TRUNCATE
| TYPE
| UNCOMMITTED
| UNICODE_LOOSE_MD5
| UNSIGNED
| UNUSED
| USAGE
| USER
| VARBINARY
| VARCHAR
| VALUE
| KUNDB_KEYSPACES
| KUNDB_RANGE_INFO
| KUNDB_SHARDS
| KUNDB_VINDEXES
| VSCHEMA_TABLES
| WRITE
| YEAR
| ZEROFILL
| CURRENT_USER

unsupported_show_keyword:
  AFTER
| AGAINST
| AVG_ROW_LENGTH
| BACKEND
| BEFORE
| BETWEEN
| BIGINT
| BINARY
| BINARY_MD5
| BIT
| BLOB
| BOOL
| BY
| CACHE
| CALL
| CASE
| CHAR
| CHARSET
| CHECKSUM
| CLOB
| COMMENT_KEYWORD
| COMPRESSION
| CONNECTION
| CONSTRAINT
| CONVERT
| CROSS
| CURRENT_DATE
| CURRENT_TIME
| CURRENT_TIMESTAMP
| DATABASE
| DATA
| DATE
| DATETIME
| DECIMAL
| DEFAULT
| DELAY_KEY_WRITE
| DELETE
| DESC
| DESCRIBE
| DIRECTORY
| DISK
| DISTINCT
| DIV
| DOUBLE
| DROP
| DUPLICATE
| EACH
| ELSE
| ENCRYPTION
| END
| ENUM
| ESCAPE
| EXISTS
| EXPANSION
| EXPLAIN
| FALSE
| FLOAT_TYPE
| FOR
| FORCE
| FROM
| FULLTEXT
| GLKJOIN
| GLOBAL
| GROUP
| HASH
| HAVING
| IF
| IGNORE
| IN
| INCEPTOR
| INNER
| INSERT
| INSERT_METHOD
| INT
| INTEGER
| INTERVAL
| INTO
| IS
| JOIN
| JSON
| KEY
| KEY_BLOCK_SIZE
| LANGUAGE
| LAST_INSERT_ID
| LEFT
| LESS
| LIKE
| LIMIT
| LIST
| LOCAL
| LOCALTIME
| LOCALTIMESTAMP
| LOCK
| LONGBLOB
| LONGTEXT
| MATCH
| MAX_ROWS
| MAXVALUE
| MEDIUMBLOB
| MEDIUMINT
| MEDIUMTEXT
| MEMORY
| MFED
| MIN_ROWS
| MOD
| MODE
| NAMES
| NATURAL
| NCHAR
| NEXT
| NOT
| NULL
| NUMERIC
| NUMERIC_STATIC_MAP
| OFFSET
| ON
| OPTIMIZE
| OR
| ORDER
| OUTER
| PACK_KEYS
| PARTITION
| PASSWORD
| PRIMARY
| QUERY
| RANGE
| REAL
| REFERENCE
| REFERENCES
| REGEXP
| RENAME
| REORGANIZE
| REPAIR
| REPLACE
| REVERSE_BITS
| RIGHT
| ROW
| ROWNUM
| SELECT
| SEPARATOR
| SEQUENCE
| SET
| SHARE
| SHOW
| SIGNED
| SMALLINT
| SPATIAL
| START
| STATS_AUTO_RECALC
| STATS_PERSISTENT
| STATS_SAMPLE_PAGES
| STATUS
| STRAIGHT_JOIN
| TABLESPACE
| TEMPORARY
| TEXT
| THAN
| THEN
| TIME
| TIMESTAMP
| TINYBLOB
| TINYINT
| TINYTEXT
| TO
| TRANSACTION
| TRIGGER
| TRUE
| UNICODE_LOOSE_MD5
| UNION
| UNIQUE
| UNSIGNED
| UNUSED
| UPDATE
| USE
| USING
| UTC_DATE
| UTC_TIME
| UTC_TIMESTAMP
| VALUES
| VARBINARY
| VARCHAR
| VIEW
| WHEN
| WHERE
| WITH
| YEAR
| ZEROFILL


openb:
  '('
  {
    if incNesting(yylex) {
      yylex.Error("max nesting level reached")
      return 1
    }
  }

closeb:
  ')'
  {
    decNesting(yylex)
  }

force_eof:
{
  forceEOF(yylex)
}

ddl_force_eof:
  {
    forceEOF(yylex)
  }
| openb
  {
    forceEOF(yylex)
  }
| reserved_sql_id
  {
    forceEOF(yylex)
  }

/* Oracle PL/SQL Procedure Support */
create_procedure_stmt:
  CREATE procedure_body
  {
    $$ = &DDL{Action: CreateStr, ProcedureBody: $2, DDLType: ProcedureDDL}
  }
| CREATE OR REPLACE procedure_body
  {
    $$ = &DDL{Action: CreateStr, ProcedureBody: $4, DDLType: ProcedureDDL, CreateOrReplace: true}
  }

procedure_body:
  PROCEDURE table_name opt_parameter_list opt_invoker_rights_clause is_or_as block ';'
  {
    $$ = &ProcedureBody{ProcName: $2, ParametersOpt: $3, InvokerRightsOpt: $4, Block: $6}
  }
| PROCEDURE table_name opt_parameter_list opt_invoker_rights_clause is_or_as call_spec ';'
  {
    $$ = &ProcedureBody{ProcName: $2, ParametersOpt: $3, InvokerRightsOpt: $4, CallSpecOpt: $6}
  }
| PROCEDURE table_name opt_parameter_list opt_invoker_rights_clause is_or_as EXTERNAL ';'
  {
    $$ = &ProcedureBody{ProcName: $2, ParametersOpt: $3, InvokerRightsOpt: $4, ExternalOpt: string($6)}
  }

drop_procedure_stmt:
  DROP PROCEDURE opt_exists table_name ';'
  {
    var exists bool
    if $3 != 0 {
      exists = true
    }

    $$ = &DDL{Action: DropStr, IfExists:exists, ProcedureBody: &ProcedureBody{ProcName:$4}, DDLType: ProcedureDDL}
  }

alter_procedure_stmt:
  ALTER procedure_alter_body
  {
    $$ = &DDL{Action: AlterStr, ProcedureAlterBody: $2, DDLType: FunctionDDL}
  }

procedure_alter_body:
  PROCEDURE table_name COMPILE opt_debug opt_compiler_parameter_list opt_reuse_settings ';'
  {
    $$ = &ProcedureAlterBody{procName: $2, OptDebug: $4, OptCompilerParameters: $5, OptReuseSettings: $6}
  }

/* Oracle PL/SQL Function Support */
create_function_stmt:
  CREATE function_body
  {
    $$ = &DDL{Action: CreateStr, FunctionBody: $2, DDLType: FunctionDDL}
  }
| CREATE OR REPLACE function_body
  {
    $$ = &DDL{Action: CreateStr, FunctionBody: $4, DDLType: FunctionDDL, CreateOrReplace: true}
  }

function_body:
  FUNCTION table_name opt_parameter_list RETURN type_spec opt_func_return_suffixs opt_pipelined is_or_as block ';'
  {
    $$ = &FunctionBody{FuncName: $2, ParametersOpt: $3, ReturnType: $5, FuncReturnSuffixs: $6, PipelinedOpt: $7, Block: $9}
  }
| FUNCTION table_name opt_parameter_list RETURN type_spec opt_func_return_suffixs opt_pipelined is_or_as call_spec ';'
  {
    $$ = &FunctionBody{FuncName: $2, ParametersOpt: $3, ReturnType: $5, FuncReturnSuffixs: $6, PipelinedOpt: $7, CallSpecOpt: $9}
  }
| FUNCTION table_name opt_parameter_list RETURN type_spec opt_func_return_suffixs pipelined_aggregate USING index_name ';'
  {
    $$ = &FunctionBody{FuncName: $2, ParametersOpt: $3, ReturnType: $5, FuncReturnSuffixs: $6, PipelinedAggregateOpt: $7, UsingImpleType: $9}
  }

drop_function_stmt:
  DROP FUNCTION opt_exists table_name ';'
  {
    var exists bool
    if $3 != 0 {
      exists = true
    }

    $$ = &DDL{Action: DropStr, IfExists:exists, FunctionBody: &FunctionBody{FuncName:$4}, DDLType: FunctionDDL}
  }

alter_function_stmt:
  ALTER function_alter_body
  {
    $$ = &DDL{Action: AlterStr, FunctionAlterBody: $2, DDLType: FunctionDDL}
  }

function_alter_body:
  FUNCTION table_name COMPILE opt_debug opt_compiler_parameter_list opt_reuse_settings ';'
  {
    $$ = &FunctionAlterBody{FuncName: $2, OptDebug: $4,  OptCompilerParameters: $5, OptReuseSettings: $6}
  }

opt_specification:
  {
    $$ = ""
  }
| PACKAGE
  {
    $$ = string($1)
  }
| BODY
  {
    $$ = string($1)
  }
| SPECIFICATION
  {
    $$ = string($1)
  }


opt_debug:
  {
    $$ = ""
  }
| DEBUG
  {
    $$ = string($1)
  }

opt_compiler_parameter_list:
  {
    $$ = nil
  }
| compiler_parameter_list
  {
    $$ = $1
  }

compiler_parameter_list:
  compiler_parameter_clause
  {
    $$ = CompilerParameterList{$1}
  }
| compiler_parameter_list compiler_parameter_clause
  {
    $$ = append($1, $2)
  }

compiler_parameter_clause:
  identifier '=' plsql_expression
  {
    $$ = &CompilerParameter{Identifier: $1, Expression: $3}
  }

opt_reuse_settings:
  {
    $$ = ""
  }
| REUSE SETTINGS
  {
    $$ = string($1) + " " + string($2)
  }

/* Oracle PL/SQL Package Support */
alter_package_stmt:
  ALTER package_alter_body
  {
    $$ = &DDL{Action: AlterStr, PackageAlterBody: $2, DDLType: ProcedureDDL}
  }

package_alter_body:
  PACKAGE table_name COMPILE opt_debug opt_specification opt_compiler_parameter_list opt_reuse_settings ';'
  {
    $$ = &PackageAlterBody{PackageName: $2, OptDebug: $4, OptSpecification: $5, OptCompilerParameters: $6, OptReuseSettings: $7}
  }

drop_package_stmt:
  DROP PACKAGE opt_exists table_name ';'
  {
    var exists bool
    if $3 != 0 {
      exists = true
    }

    $$ = &DDL{Action: DropStr, IfExists:exists, PackageSpec: &PackageSpec{PackageName:$4}, DDLType: PackageDDL}
  }
| DROP PACKAGE BODY opt_exists table_name ';'
  {
    var exists bool
    if $4 != 0 {
      exists = true
    }

    $$ = &DDL{Action: DropStr, IfExists:exists, PackageBody: &PackageBody{PackageName:$5}, DDLType: PackageDDL}
  }

create_package_stmt:
  CREATE PACKAGE package_spec
  {
    $$ = &DDL{Action: CreateStr, PackageSpec: $3, DDLType: PackageDDL}
  }
| CREATE OR REPLACE PACKAGE package_spec
  {
    $$ = &DDL{Action: CreateStr, PackageSpec: $5, DDLType: PackageDDL, CreateOrReplace: true}
  }

package_spec:
  table_name opt_invoker_rights_clause is_or_as END opt_name ';'
  {
    $$ = &PackageSpec{}
    $$.PackageName = $1
    $$.InvokerRightsOpt = $2
  }
| table_name opt_invoker_rights_clause is_or_as package_obj_spec_list END opt_name ';'
  {
    $$ = $4
    $$.PackageName = $1
    $$.InvokerRightsOpt = $2
  }

pl_declaration:
  procedure_spec
  {
    $$ = $1
  }
| function_spec
  {
    $$ = $1
  }
| variable_declaration
  {
    $$ = $1
  }
| subtype_declaration
  {
    $$ = $1
  }
| cursor_declaration
  {
    $$ = $1
  }
| exception_declaration
  {
    $$ = $1
  }
| type_declaration
  {
    $$ = $1
  }

package_obj_spec_list:
  pl_declaration
  {
    $$ = &PackageSpec{}
    $$.PLDeclaration = append($$.PLDeclaration, $1)
  }
| package_obj_spec_list pl_declaration
  {
    $$.PLDeclaration = append($$.PLDeclaration, $2)
  }

create_package_body_stmt:
  CREATE PACKAGE BODY package_body
  {
    $$ = &DDL{Action: CreateStr, PackageBody: $4, DDLType: PackageDDL}
  }
| CREATE OR REPLACE PACKAGE BODY package_body
  {
    $$ = &DDL{Action: CreateStr, PackageBody: $6, DDLType: PackageDDL, CreateOrReplace: true}
  }

package_body:
  table_name is_or_as END opt_name ';'
  {
    $$ = &PackageBody{}
    $$.PackageName = $1
  }
| table_name is_or_as package_obj_body_list END opt_name ';'
  {
    $$ = $3
    $$.PackageName = $1
    }
| table_name is_or_as package_obj_body_list body ';'
  {
    $$ = $3
    $$.PackageName = $1
    $$.SuffixBodyOpt = $4
    }

pl_package_body:
  procedure_spec
  {
    $$ = $1
  }
| function_spec
  {
    $$ = $1
  }
| variable_declaration
  {
    $$ = $1
  }
| subtype_declaration
  {
    $$ = $1
  }
| cursor_declaration
  {
    $$ = $1
  }
| exception_declaration
  {
    $$ = $1
  }
| type_declaration
  {
    $$ = $1
  }
| procedure_definition
  {
    $$ = $1
  }
| function_definition
  {
    $$ = $1
  }

package_obj_body_list:
  pl_package_body
  {
    $$ = &PackageBody{}
    $$.PLPackageBody = append($$.PLPackageBody, $1)
  }
| package_obj_body_list pl_package_body
  {
    $$.PLPackageBody = append($$.PLPackageBody, $2)
  }

/* PL/SQL Package Elements Declarations and Bodys*/

procedure_spec:
  PROCEDURE table_name opt_parameter_list ';'
  {
    $$ = &ProcedureSpec{ProcName: $2, ParametersOpt: $3}
  }

variable_declaration:
  ID opt_constant type_spec opt_not_null opt_default_value_part ';'
  {
    $$ = &VariableDeclaration{VariableName: string($1), ConstraintOpt: string($2), TypeSpec: $3, NullOpt: string($4), DefaultValuePartOpt: $5}
  }

subtype_declaration:
  SUBTYPE ID IS type_spec opt_subtype_range opt_not_null ';'
  {
    $$ = &SubTypeDeclaration{SubTypeName: string($2), Type: $4, RangeOpt: $5, NullOpt: $6}
  }

cursor_declaration:
  CURSOR ID opt_parameter_spec_list opt_cursor_return opt_is_select ';'
  {
    $$ = &CursorDeclaration{CursorName: string($2), ParameterSpecsOpt: $3, ReturnTypeOpt: $4, IsStatementOpt: $5}
  }

exception_declaration:
  ID EXCEPTION ';'
  {
    $$ = &ExceptionDeclaration{ExceptionName: string($1)}
  }

type_declaration:
  TYPE ID IS table_type_def ';'
  {
    $$ = &TypeDeclaration{TypeName: string($2), TableTypeOpt: $4}
  }
| TYPE ID IS varray_type_def ';'
  {
    $$ = &TypeDeclaration{TypeName: string($2), VarrayTypeOpt: $4}
  }
| TYPE ID IS record_type_def ';'
  {
    $$ = &TypeDeclaration{TypeName: string($2), RecordTypeOpt: $4}
  }
| TYPE ID IS cursor_type_def ';'
  {
    $$ = &TypeDeclaration{TypeName: string($2), CursorTypeOpt: $4}
  }

procedure_definition:
  PROCEDURE table_name opt_parameter_list is_or_as block ';'
  {
    $$ = &ProcedureDefinition{ProcName: $2, ParametersOpt: $3, Block: $5}
  }
| PROCEDURE table_name opt_parameter_list is_or_as call_spec ';'
  {
    $$ = &ProcedureDefinition{ProcName: $2, ParametersOpt: $3, CallSpecOpt: $5}
  }
| PROCEDURE table_name opt_parameter_list is_or_as EXTERNAL ';'
  {
    $$ = &ProcedureDefinition{ProcName: $2, ParametersOpt: $3, ExternalOpt: string($5)}
  }

function_spec:
  FUNCTION table_name opt_parameter_list RETURN type_spec opt_func_return_suffixs ';'
  {
    $$ = &FunctionSpec{FullName: $2, ParametersOpt: $3, ReturnType: $5, ReturnOpt: $6}
  }

function_definition:
  FUNCTION table_name opt_parameter_list RETURN type_spec opt_func_return_suffixs opt_pipelined is_or_as block ';'
  {
    $$ = &FunctionDefinition{FuncName: $2, ParametersOpt: $3, ReturnType: $5, FuncReturnSuffixs: $6, PipelinedOpt: $7, Block: $9}
  }
| FUNCTION table_name opt_parameter_list RETURN type_spec opt_func_return_suffixs opt_pipelined is_or_as call_spec ';'
  {
    $$ = &FunctionDefinition{FuncName: $2, ParametersOpt: $3, ReturnType: $5, FuncReturnSuffixs: $6, PipelinedOpt: $7, CallSpecOpt: $9}
  }
| FUNCTION table_name opt_parameter_list RETURN type_spec opt_func_return_suffixs pipelined_aggregate USING index_name ';'
  {
    $$ = &FunctionDefinition{FuncName: $2, ParametersOpt: $3, ReturnType: $5, FuncReturnSuffixs: $6, PipelinedAggregateOpt: $7, UsingImpleType: $9}
  }

/* Common PL/SQL Named Elements */
opt_func_return_suffixs:
  {
    $$ = nil
  }
| func_return_suffixs
  {
    $$ = $1
  }

func_return_suffixs:
  func_return_suffix
  {
    $$ = FuncReturnSuffixList{$1}
  }
| func_return_suffixs func_return_suffix
  {
    $$ = append($1, $2)
  }

func_return_suffix:
  invoker_rights_clause
  {
    $$ = $1
  }
| parallel_enable_clause
  {
    $$ = $1
  }
| result_cache_clause
  {
    $$ = $1
  }
| deterministic
  {
    $$ = $1
  }

invoker_rights_clause:
  AUTHID CURRENT_USER
  {
    $$ = &InvokerRightsClause{KeyWord: string($1) + " " + string($2)}
  }
| AUTHID DEFINER
  {
    $$ = &InvokerRightsClause{KeyWord: string($1) + " " + string($2)}
  }

parallel_enable_clause:
  PARALLEL_ENABLE opt_partition_by_clause
  {
    $$ = &ParallelEnableClause{PartitionByClause: $2}
  }

result_cache_clause:
  RESULT_CACHE opt_relies_on_part
  {
    $$ = &ResultCacheClause{ReliesOnPart: $2}
  }

opt_relies_on_part:
  {
    $$ = nil
  }
| relies_on_part
 {
   $$ = $1
 }

relies_on_part:
  RELIES_ON '(' table_view_names ')'
  {
    $$ = &ReliesOnPart{TableViewNames: $3}
  }

opt_for:
  {
    $$ = ""
  }
| FOR
  {
    $$ = string($1)
  }

partition_extension:
  SUBPARTITION opt_for '(' opt_plsql_expressions ')'
  {
    $$ = &PartitionExtension{PartitionMode: string($1), ForOpt: $2, ExpressionsOpt: $4}
  }
| PARTITION opt_for '(' opt_plsql_expressions ')'
  {
    $$ = &PartitionExtension{PartitionMode: string($1), ForOpt: $2, ExpressionsOpt: $4}
  }

opt_id_expression:
  {
    $$ = nil
  }
| '.' id_expression
  {
    $$ = $2
  }

table_view_name:
  identifier opt_id_expression
  {
    $$ = &TableViewName{Identifier: $1, IdExpressionOpt: $2}
  }
| identifier opt_id_expression '@' identifier
  {
    $$ = &TableViewName{Identifier: $1, IdExpressionOpt: $2, LinkNameOpt: $4}
  }
| identifier opt_id_expression partition_extension
  {
    $$ = &TableViewName{Identifier: $1, IdExpressionOpt: $2, PartitionExptenOpt: $3}
  }

table_view_names:
  table_view_name
  {
    $$ = TableViewNameList{$1}
  }
| table_view_names ',' table_view_name
  {
    $$ = append($1, $3)
  }

deterministic:
  DETERMINISTIC
  {
    $$ = &Deterministic{KeyWord: string($1)}
  }

paren_column_list:
  '(' column_list ')'
  {
    $$ = &ParenColumnList{ColumnList: $2}
  }

opt_stream_clause:
  {
    $$ = nil
  }
| stream_clause
  {
    $$ = $1
  }

stream_clause:
  ORDER plsql_expression BY paren_column_list
  {
    $$ = &StreamingClause{Type: string($1), Expression: $2, ParenColumnList: $4}
  }
| CLUSTER plsql_expression BY paren_column_list
  {
    $$ = &StreamingClause{Type: string($1), Expression: $2, ParenColumnList: $4}
  }

opt_partition_by_clause:
  {
    $$ = nil
  }
| partition_by_clause
  {
    $$ = $1
  }

hash_range:
  HASH
  {
    $$ = string($1)
  }
| RANGE
  {
    $$ = string($1)
  }

partition_by_clause:
  '(' PARTITION plsql_expression BY ANY ')' opt_stream_clause
  {
    $$ = &PartitionByClause{Expression: $3, AnyOpt: string($5), StreamingClause: $7}
  }
| '(' PARTITION plsql_expression BY hash_range paren_column_list ')' opt_stream_clause
  {
    $$ = &PartitionByClause{Expression: $3, Type: string($5), ParenColumnList: $6, StreamingClause: $8}
  }

pipelined_aggregate:
  PIPELINED
  {
    $$ = string($1)
  }
| AGGREGATE
  {
    $$ = string($1)
  }

opt_pipelined:
  {
    $$ = ""
  }
| PIPELINED
  {
    $$ = string($1)
  }

body:
  BEGIN seq_of_statements END opt_name
  {
    $$ = &Body{SeqOfStatement: $2, EndNameOpt: string($4)}
  }
| BEGIN seq_of_statements exception_handler_list END opt_name
  {
    $$ = &Body{SeqOfStatement: $2, ExceptionHandlersOpt: $3, EndNameOpt: string($5)}
  }

opt_dynamic_return:
  {
    $$ = nil
  }
| dynamic_return
  {
    $$ = $1
  }

dynamic_return:
  RETURN into_clause
  {
    $$ = &DynamicReturn{ReturnName: string($1), IntoClause: $2}
  }
| RETURNING into_clause
  {
    $$ = &DynamicReturn{ReturnName: string($1), IntoClause: $2}
  }

using_return:
  using_clause opt_dynamic_return
  {
    $$ = &UsingReturn{UsingClause: $1, ReturnOpt: $2}
  }

variable_name:
  INTRODUCER id_expression_list id_expression_list
  {
    $$ = &VariableName{CharSetNameOpt: $2, Name: $3}
  }
| id_expression_list
  {
    $$ = &VariableName{Name: $1}
  }

variable_names:
  variable_name
  {
    $$ = VariableNameList{$1}
  }
| variable_names ',' variable_name
  {
    $$ = append($1, $3)
  }

opt_into_clause:
  {
    $$ = nil
  }
| into_clause
  {
    $$ = $1
  }

into_clause:
  BULK COLLECT INTO variable_names
  {
    $$ = &IntoClause{BulkOpt: string($1) + string($2), VariableNames: $4}
  }
| INTO variable_names
  {
    $$ = &IntoClause{VariableNames: $2}
  }

opt_scope:
  {
    $$ = ""
  }
| IN
  {
    $$ = string($1)
  }
| IN OUT
  {
    $$ = string($1) + " " +string($2)
  }
| OUT
  {
    $$ = string($1)
  }



using_element:
  opt_scope sql_id
  {
    $$ =&UsingElement{ParameterOpt: $1, SelectElement: $2.String()}
  }
| opt_scope sql_id AS col_alias
  {
    $$ =&UsingElement{ParameterOpt: $1, SelectElement: $2.String(), AliasOpt: $4.String()}
  }
| opt_scope STRING
  {
    $$ =&UsingElement{ParameterOpt: $1, SelectElement: "'" + string($2) + "'"}
  }
| opt_scope STRING AS col_alias
  {
    $$ =&UsingElement{ParameterOpt: $1, SelectElement: "'" + string($2) + "'", AliasOpt: $4.String()}
  }

using_elements:
  using_element
  {
    $$ = UsingElementList{$1}
  }
| using_elements ',' using_element
  {
    $$ = append($1, $3)
  }

using_clause:
  USING using_elements
  {
    $$ = &UsingClause{UsingElementOpt: $2}
  }
| USING '*'
  {
    $$ = &UsingClause{AllOpt: "*"}
  }

into_using:
  into_clause
  {
    $$ = &IntoUsing{IntoClause: $1}
  }
| into_clause using_clause
 {
    $$ = &IntoUsing{IntoClause: $1, UsingOpt: $2}
 }

execute_immediate:
  EXECUTE IMMEDIATE plsql_expression into_using
  {
    $$ = &ExecuteImmediate{StmtStr: $3, Suffix: $4}
  }
| EXECUTE IMMEDIATE plsql_expression using_return
  {
    $$ = &ExecuteImmediate{StmtStr: $3, Suffix: $4}
  }
| EXECUTE IMMEDIATE plsql_expression dynamic_return
  {
    $$ = &ExecuteImmediate{StmtStr: $3, Suffix: $4}
  }

open_statement:
  OPEN variable_name plsql_expressions
  {
    $$ = &OpenStatement{CursorName: $2, PlsqlExpressionsOpt: $3}
  }
| OPEN variable_name
  {
    $$ = &OpenStatement{CursorName: $2}
  }

fetch_statement:
  FETCH column_name INTO variable_names
  {
    $$ = &FetchStatement{CursorName: $2, IntoOpt: $4}
  }
| FETCH column_name BULK COLLECT INTO variable_names
  {
    $$ = &FetchStatement{CursorName: $2, BulkIntoOpt: $6}
  }

open_for_statement:
  OPEN variable_name FOR plsql_expression using_clause
  {
    $$ = &OpenForStatement{VariableName: $2, ExpressionOpt: $4, UsingClause: $5}
  }
| OPEN variable_name FOR plsql_expression
  {
    $$ = &OpenForStatement{VariableName: $2, ExpressionOpt: $4}
  }
| OPEN variable_name FOR base_select opt_order_by opt_limit opt_lock  using_clause
  {
    sel := $4.(*Select)
    sel.OrderBy = $5
    sel.Limit = $6
    sel.Lock = $7
    $$ = &OpenForStatement{VariableName: $2, SelStatementOpt: sel, UsingClause: $8}
  }
| OPEN variable_name FOR base_select opt_order_by opt_limit opt_lock
  {
    sel := $4.(*Select)
    sel.OrderBy = $5
    sel.Limit = $6
    sel.Lock = $7
    $$ = &OpenForStatement{VariableName: $2, SelStatementOpt: sel}
  }

cursor_manu_statement:
  CLOSE column_name
  {
    $$ = &CursorManuStatement{CloseStatement: $2.String()}
  }
| open_statement
  {
    $$ = &CursorManuStatement{OpenStatement: $1}
  }
| fetch_statement
  {
    $$ = &CursorManuStatement{FetchStatement: $1}
  }
| open_for_statement
  {
    $$ = &CursorManuStatement{OpenForStatement: $1}
  }

only_write:
  ONLY
  {
    $$ = string($1)
  }
| WRITE
  {
    $$ = string($1)
  }

opt_name_quoted:
  {
    $$ = ""
  }
| NAME quoted_string
  {
    $$ = $2
  }

serializable_read_committed:
  SERIALIZABLE
  {
    $$ = string($1)
  }
| READ COMMITTED
  {
    $$ = string($1) + " " + string($2)
  }

set_trans_command:
  SET TRANSACTION READ only_write opt_name_quoted
  {
    $$ = &SetTransCommand{ReadOpt: $4, QuotedString: $5}
  }
| SET TRANSACTION ISOLATION LEVEL serializable_read_committed opt_name_quoted
  {
    $$ = &SetTransCommand{IsolationOpt: $5, QuotedString: $6}
  }
| SET TRANSACTION USE ROLLBACK SEGMENT sql_id opt_name_quoted
  {
    $$ = &SetTransCommand{UseOpt: $6, QuotedString: $7}
  }

constraint_name:
  sql_id id_expression_list opt_link_name
  {
    $$ = &ConstraintName{Prefix: $1, Expressions: $2, LinkNameOpt: $3}
  }

constraint_names:
  constraint_name
  {
    $$ = ConstraintNameList{$1}
  }
| constraint_names ',' constraint_name
  {
    $$ = append($1, $3)
  }

constraint_constraints:
  CONSTRAINT
  {
    $$ = string($1)
  }
| CONSTRAINTS
  {
    $$ = string($1)
  }

immediate_deferred:
  IMMEDIATE
  {
    $$ = string($1)
  }
| DEFERRED
  {
    $$ = string($1)
  }

set_const_command:
  SET constraint_constraints ALL immediate_deferred
  {
    $$ = &SetConstCommand{ConstraintOpt: $2, AllOpt: string($3), SuffixOpt: $4}
  }
| SET constraint_constraints constraint_names immediate_deferred
  {
    $$ = &SetConstCommand{ConstraintOpt: $2, ConstraintNamesOpt: $3, SuffixOpt: $4}
  }

wait_nowait:
  WAIT
  {
    $$ = string($1)
  }
| NOWAIT
  {
    $$ = string($1)
  }

immediate_batch:
  IMMEDIATE
  {
    $$ = string($1)
  }
| BATCH
  {
    $$ = string($1)
  }

opt_write_clause:
  WRITE wait_nowait immediate_batch
  {
    $$ = string($1) + " " + string($2) + " " + string($3)
  }

plsql_commit_statement:
//  COMMIT opt_work COMMENT plsql_expression opt_write_clause  todo COMMENT TOKEN
//  {
//    $$ = &CommitStatement{WorkOpt: $2, CommentOpt: $4, WriteClause: $5}
//  }
  COMMIT opt_work FORCE CORRUPT_XID plsql_expression opt_write_clause
  {
    $$ = &CommitStatement{WorkOpt: $2, ForceExprOpt: $5, WriteClause: $6}
  }
| COMMIT opt_work FORCE plsql_expressions opt_write_clause
  {
    $$ = &CommitStatement{WorkOpt: $2, ForceExprsOpt: $4, WriteClause: $5}
  }
| COMMIT opt_work FORCE CORRUPT_XID_ALL opt_write_clause
  {
    $$ = &CommitStatement{WorkOpt: $2, ForceKeyword: string($4), WriteClause: $5}
  }

opt_work:
  {
    $$ = ""
  }
| WORK
  {
    $$ = string($1)
  }

opt_savepoint:
  {
    $$ = ""
  }
| SAVEPOINT
  {
    $$ = string($1)
  }

quoted_string:
  STRING
  {
    $$ = "'" + string($1) + "'"
  }

identifier:
  INTRODUCER id_expression_list id_expression
  {
    $$ = &Identifier{CharsetNameOpt: $2, IdExpression: $3}
  }
| id_expression
  {
    $$ = &Identifier{IdExpression: $1}
  }

plsql_rollback_statement:
  ROLLBACK opt_work TO opt_savepoint identifier
  {
    $$ = &RollbackStatement{WorkOpt: $2, SavePointOpt: $4, SavePointName: $5}
  }
| ROLLBACK opt_work FORCE quoted_string
  {
    $$ = &RollbackStatement{WorkOpt: $2, SavePointOpt: $4, QuotedString: $4}
  }
| ROLLBACK WORK
  {
    $$ = &RollbackStatement{WorkOpt: string($2)}
  }

save_point_statement:
  SAVEPOINT identifier
  {
    $$ = &SavePointStatement{SavePointName: $2}
  }

trans_ctrl_statement:
  set_trans_command
  {
    $$ = &TransCtrlStatement{SetTransCommand: $1}
  }
| set_const_command
  {
    $$ = &TransCtrlStatement{SetConstCommand: $1}
  }
| plsql_commit_statement
  {
    $$ = &TransCtrlStatement{CommitStatement: $1}
  }
| plsql_rollback_statement
  {
    $$ = &TransCtrlStatement{RollbackStatement: $1}
  }
| save_point_statement
  {
    $$ = &TransCtrlStatement{SavePointStatement: $1}
  }

dml_tatement:
  select_statement
  {
    $$ = &DmlStatement{SelectStatement: $1}
  }
| update_statement
  {
    $$ = &DmlStatement{UpdateStatement: $1}
  }
| delete_statement
  {
    $$ = &DmlStatement{DeleteStatement: $1}
  }
| insert_statement
  {
    $$ = &DmlStatement{InsertStatement: $1}
  }
| explain_statement
  {
    $$ = &DmlStatement{ExplainStatement: $1}
  }

sql_statement:
  execute_immediate ';'
  {
    $$ = &SqlStatement{ExecuteImmediate: $1}
  }
| cursor_manu_statement	';'
  {
    $$ = &SqlStatement{CursorManuStatement: $1}
  }
| trans_ctrl_statement ';'
  {
    $$ = &SqlStatement{TransCtrlStatement: $1}
  }
| dml_tatement ';'
  {
    $$ = &SqlStatement{DmlStatement: $1}
  }


simple_case_when_part:
  WHEN plsql_expression THEN seq_of_statements
  {
    $$ = &SimpleCaseWhenPart{WhenCondition: $2,  SeqOfStatementOpt: $4}
  }
//| WHEN plsql_expression THEN plsql_expression ';'
//  {
//    $$ = &SimpleCaseWhenPart{WhenCondition: $2,  ExpressionOpt: $4}
//  }

simple_case_when_parts:
  simple_case_when_part
  {
    $$ = SimpleCaseWhenPartList{$1}
  }
| simple_case_when_parts simple_case_when_part
  {
    $$ = append($1, $2)
  }

opt_case_else_part:
  ELSE seq_of_statements
  {
    $$ = &CaseElsePart{SeqOfStatementOpt: $2}
  }
//| ELSE plsql_expression ';'
//  {
//    $$ = &CaseElsePart{ExpressionOpt: $2}
//  }

opt_simple_case_statement:
  id_expression CASE plsql_expression simple_case_when_parts opt_case_else_part END CASE opt_label_name
  {
    $$ = &SimpleCaseStatement{StartLabelNameOpt: $1.String(), CaseCondition: $3, CaseWhenPart: $4, CaseElsePartOpt: $5, EndLabelName: $8}
  }
| CASE plsql_expression simple_case_when_parts opt_case_else_part END CASE opt_label_name
  {
    $$ = &SimpleCaseStatement{CaseCondition: $2, CaseWhenPart: $3, CaseElsePartOpt: $4, EndLabelName: $7}
  }

search_case_when_part:
  WHEN plsql_expression THEN seq_of_statements
  {
    $$ = &SearchCaseWhenPart{WhenCondition: $2,  SeqOfStatementOpt: $4}
  }
//| WHEN plsql_expression THEN plsql_expression ';'
//  {
//    $$ = &SearchCaseWhenPart{WhenCondition: $2,  ExpressionOpt: $4}
//  }

search_case_when_parts:
  search_case_when_part
  {
    $$ = SearchCaseWhenPartList{$1}
  }
| search_case_when_parts search_case_when_part
  {
    $$ = append($1, $2)
  }


opt_search_case_statement:
  id_expression CASE search_case_when_parts opt_case_else_part END CASE opt_label_name
  {
    $$ = &SearchCaseStatement{StartLabelNameOpt: $1.String(), CaseWhenPart: $3, CaseElsePartOpt: $4, EndLabelName: $7}
  }
| CASE search_case_when_parts opt_case_else_part END CASE opt_label_name
  {
    $$ = &SearchCaseStatement{CaseWhenPart: $2, CaseElsePartOpt: $3, EndLabelName: $6}
  }

case_statement:
  opt_simple_case_statement ';'
  {
    $$ = &CaseStatement{SimpleCaseOpt: $1}
  }
| opt_search_case_statement ';'
  {
    $$ = &CaseStatement{SearchCaseOpt: $1}
  }


opt_label_name:
  {
    $$ = ""
  }
| id_expression
  {
    $$ = $1.String()
  }

plsql_expressions:
  plsql_expression
  {
    $$ = PlsqlExpressions{$1}
  }
| plsql_expressions ',' plsql_expression
  {
    $$ = append($1, $3)
  }

 opt_plsql_expressions:
  {
    $$ = nil
  }
| plsql_expressions
  {
    $$ = $1
  }

lower_bound:
  value
  {
    $$ = $1
  }
| column_name
  {
    $$ = $1
  }

upper_bound:
  value
  {
    $$ = $1
  }
| column_name
  {
    $$ = $1
  }

loop_param:
  column_name IN REVERSE lower_bound DOUBLE_POINT value_expression
  {
    $$ = &IndexLoopParam{IndexName: $1, ReverseOpt: string($3), LowBound: $4, upperBound: $6}
  }
| column_name IN lower_bound DOUBLE_POINT value_expression
  {
    $$ = &IndexLoopParam{IndexName: $1, LowBound: $3, upperBound: $5}
  }
| column_name IN '(' column_name ')'
  {
    $$ = &RecordLoopParam{RecordName: $1, CursorNameOpt: $4}
  }
| column_name IN '(' column_name '(' plsql_expressions ')' ')'
  {
    $$ = &RecordLoopParam{RecordName: $1, CursorNameOpt: $4, PlsqlExpressionsOpt: $6}
  }
| column_name IN '(' base_select opt_order_by opt_limit opt_lock ')'
  {
    sel := $4.(*Select)
    sel.OrderBy = $5
    sel.Limit = $6
    sel.Lock = $7
    $$ = &RecordLoopParam{RecordName: $1, SelectStatementOpt: sel}
  }

loop_statement:
  WHILE condition LOOP seq_of_statements END LOOP opt_label_name ';'
  {
   $$ = &LoopStatement{WhileConditionOpt: $2, SeqOfStatement: $4, EndLabelNameOpt: $7}
  }
| LOOP seq_of_statements END LOOP opt_label_name ';'
  {
    $$ = &LoopStatement{SeqOfStatement: $2, EndLabelNameOpt: $5}
  }
| FOR loop_param LOOP seq_of_statements END LOOP opt_label_name ';'
  {
    $$ = &LoopStatement{ForLoopParamOpt: $2, SeqOfStatement: $4, EndLabelNameOpt: $7}
  }

//general_element:
//  general_element_part
//  {
//    $$ = GeneralElement{$1}
//  }
//| general_element '.' general_element_part
//  {
//    $$ = append($1, $3)
//  }
//
//opt_charset_name:
//  {
//    $$ = ""
//  }
//| INTRODUCER id_expression_list
//  {
//    $$ = $2
//  }
//
//general_element_part:
//  opt_charset_name id_expression_list opt_link_name opt_select_expression_list
//  {
//    $$ = &GeneralElementPart{CharSetNameOpt: $1, IdExpressions: $2, LinkNameOpt: $3, FuncArgumentOpt: $4}
//  }
//
//opt_general_element_parts:
//  {
//    $$ = nil
//  }
//| '.' general_element_part
// {
//    $$ = GeneralElementParList{$2}
//  }
//| general_element '.' general_element_part
//  {
//    $$ = append($1, $3)
//  }
//
//bind_variable_prefix:
//  BINDVAR
//  {
//    $$ = string($1)
//  }
//| UNSIGNED_INTEGER
//  {
//    $$ = string($1)
//  }
//
//bind_variable:
//  bind_variable_prefix opt_general_element_parts
//  {
//    $$ = &BindVariable{BindVarPrefix: $1, GeneralElementPartList: $2}
//  }

assignment_statement:
  identifier ASSIGN value_expression ';'
  {
    $$ = &AssignmentStatement{GeneralElementOpt: $1, Expression: $3}
  }

continue_statement:
  CONTINUE opt_name ';'
  {
    $$ = &ContinueStatement{LableNameOpt: $2}
  }
| CONTINUE opt_name WHEN condition ';'
  {
    $$ = &ContinueStatement{LableNameOpt: $2, ConditionOpt: $4}
  }

exit_statement:
  EXIT opt_name ';'
  {
    $$ = &ExitStatement{LableNameOpt: $2}
  }
| EXIT opt_name WHEN condition ';'
  {
    $$ = &ExitStatement{LableNameOpt: $2, ConditionOpt: $4}
  }

go_to_statement:
  GOTO sql_id ';'
  {
    $$ = &GoToStatement{LableName: $2.String()}
  }

else_part:
  ELSE seq_of_statements
  {
    $$ = &ElsePart{SeqOfStatement: $2}
  }

else_if:
  ELSIF condition THEN seq_of_statements
  {
    $$ = &ElseIf{Condition: $2, SeqOfStatement: $4}
  }

else_if_list:
  else_if
  {
    $$ = ElseIfList{$1}
  }
| else_if_list else_if
  {
    $$ = append($1, $2)
  }

if_statement:
  IF condition THEN seq_of_statements else_if_list else_part END IF ';'
  {
    $$ = &IfStatement{Condition: $2, SeqOfStatement: $4, ElseIfsPart: $5, ElsePartOpt: $6}
  }
| IF condition THEN seq_of_statements else_part END IF ';'
  {
    $$ = &IfStatement{Condition: $2, SeqOfStatement: $4, ElsePartOpt: $5}
  }
| IF condition THEN seq_of_statements else_if_list END IF ';'
  {
    $$ = &IfStatement{Condition: $2, SeqOfStatement: $4, ElseIfsPart: $5}
  }
| IF condition THEN seq_of_statements END IF ';'
  {
    $$ = &IfStatement{Condition: $2, SeqOfStatement: $4}
  }

null_statement:
  NULL ';'
  {
    $$ = &NullStatement{Null: string($1)}
  }

raise_statement:
  RAISE sql_id ';'
  {
    $$ = &RaiseStatement{ExcetionNameOpt: $2.String()}
  }
| RAISE ';'
  {
    $$ = &RaiseStatement{ExcetionNameOpt: ""}
  }

return_statement:
  RETURN ';'
  {
    $$ = &ReturnStatement{ExpressionOpt: nil}
  }
| RETURN plsql_expression ';'
  {
    $$ = &ReturnStatement{ExpressionOpt: $2}
  }

function_call_statement:
  CALL proc_or_func_name '(' opt_select_expression_list ')' ';'
  {
    $$ = &FunctionCallStatement{RoutineName: $2, ArgumentsOpt: $4}
  }

pipe_row_statement:
  PIPE ROW '(' plsql_expression ')' ';'
  {
    $$ = &PipeRowStatement{Expression: $4}
  }

label_declaration:
  SHIFT_LEFT id_expression SHIFT_RIGHT
  {
    $$ = &LabelDeclaration{LabelName: $2.String()}
  }

between_bound:
  BETWEEN lower_bound AND upper_bound
  {
    $$ = &BetweenBound{LowBound: $2, UpperBound: $4}
  }

opt_between_bound:
  {
    $$ = nil
  }
| between_bound
  {
    $$ = $1
  }

index_name:
  identifier
  {
    $$ = &IndexName{Identifier: $1}
  }
| identifier '.' id_expression
  {
    $$ = &IndexName{Identifier: $1, IdExpressionOpt: $3}
  }

bounds_clause:
  lower_bound DOUBLE_POINT upper_bound
  {
    $$ = &BoundsClause{LowBound: $1, UpperBound: $3}
  }
| INDICES OF index_name opt_between_bound
  {
    $$ = &BoundsClause{IndicesOfOpt: $3, BetweenBoundOpt: $4}
  }
| VALUES OF index_name
  {
    $$ = &BoundsClause{ValuesOf: $3}
  }

forall_statement:
  FORALL index_name IN bounds_clause sql_statement
  {
    $$ = &ForAllStatement{IndexName: $2, BoundsClause: $4, SqlStatement: $5}
  }
| FORALL index_name IN bounds_clause sql_statement SAVE EXCEPTIONS ';'
  {
    $$ = &ForAllStatement{IndexName: $2, BoundsClause: $4, SqlStatement: $5, SaveExceptionsOpt: string($6) + " " +string($7)}
  }

pl_statement:
  assignment_statement
  {
    $$ = $1
  }
| continue_statement
  {
    $$ = $1
  }
| exit_statement
  {
    $$ = $1
  }
| go_to_statement
  {
    $$ = $1
  }
| if_statement
  {
    $$ = $1
  }
| null_statement
  {
    $$ = $1
  }
| raise_statement
  {
    $$ = $1
  }
| return_statement
  {
    $$ = $1
  }
| function_call_statement
  {
    $$ = $1
  }
| pipe_row_statement
  {
    $$ = $1
  }
| loop_statement
  {
    $$ = $1
  }
| case_statement
  {
    $$ = $1
  }
| sql_statement
  {
    $$ = $1
  }
| forall_statement
  {
    $$ = $1
  }
| label_declaration
  {
    $$ = $1
  }

seq_of_statements:
  pl_statement
  {
    $$ = &SeqOfStatement{}
    $$.PlStatements = append($$.PlStatements, $1)
  }
| seq_of_statements pl_statement
  {
    $$.PlStatements = append($$.PlStatements, $2)
  }

exception_handler:
  EXCEPTION WHEN sql_id THEN seq_of_statements
  {
    $$ = &ExceptionHandler{ExceptionName: $3.String(), SeqOfStatement: $5}
  }

exception_handler_list:
  exception_handler
  {
    $$ = ExceptionHandlerList{$1}
  }
| exception_handler_list exception_handler
  {
    $$ = append($1, $2)
  }

declare_spec:
  pl_declaration
  {
    $$ = &DeclareSpec{}
    $$.PLDeclaration = append($$.PLDeclaration, $1)
  }
| declare_spec pl_declaration
  {
    $$.PLDeclaration = append($$.PLDeclaration, $2)
  }

opt_declare:
  {
    $$ = ""
  }
| DECLARE
  {
    $$ = string($1)
  }

block:
  opt_declare declare_spec body
  {
    $$ = &Block{DeclareSpecsOpt: $2, Body: $3}
  }
| opt_declare body
  {
    $$ = &Block{Body: $2}
  }

c_spec:
  C_LETTER force_eof
  {
    $$ = nil
  }
  // todo

call_spec:
  LANGUAGE JAVA NAME STRING
  {
    $$ = &CallSpec{JavaSpec: string($2) + " " + string($3) + " " + string($4)}
  }
| LANGUAGE c_spec
  {
    $$ = &CallSpec{CSpec: $2}
  }

id_expression_list:
  id_expression
  {
    $$ = IdExpressionList{$1}
  }
| id_expression_list '.' id_expression
  {
    $$ = append($1, $3)
  }

opt_link_name:
  {
    $$ = ""
  }
| '@' ID
  {
    $$ = "@" + string($2)
  }

opt_table_index:
  {

  }
| ID '=' INDEX BY type_spec
  {

  }
  // todo

table_type_def:
  TABLE OF type_spec opt_table_index opt_not_null
  {
    $$ = &TableTypeDef{Type: $3, TableIndexOpt: $4, NotNullOpt: $5}
  }

opt_varray_prefix:
  VARRAY
  {
   $$ = string($1)
  }
| VARYING ARRAY
  {
   $$ = string($1) + " " + string($2)
  }

varray_type_def:
  opt_varray_prefix '(' plsql_expression ')' OF type_spec opt_not_null
  {
    $$ = &VarrayTypeDef{VarrayPrefix: $1, Expression: $3, Type: $6, NotNullOpt: $7}
  }

field_spec:
  table_name opt_type_spec opt_not_null opt_default_value_part
  {
    $$ = &FieldSpec{columnName: $1, TypeOpt: $2, NotNullOpt: $3, DefaultValueOpt: $4}
  }

field_spec_list:
  field_spec
  {
    $$ = FieldSpecList{$1}
  }
| field_spec_list ',' field_spec
  {
    $$ = append($1, $3)
  }

record_type_def:
  RECORD '(' field_spec_list ')'
  {
    $$ = &RecordTypeDef{FieldSpecs: $3}
  }

cursor_type_def:
  REF CURSOR opt_cursor_return
  {
    $$ = &CursorTypeDef{ReturnTypeOpt: $3}
  }

cursor_expression:
  CURSOR subquery
  {
    $$ = &CursorExpression{SubQuery: $2}
  }

opt_percent:
  PERCENT_ISOPEN
  {
    $$ = string($1)
  }
| PERCENT_FOUND
  {
    $$ = string($1)
  }
| PERCENT_NOTFOUND
  {
    $$ = string($1)
  }
| PERCENT_ROWCOUNT
  {
    $$ = string($1)
  }

logical_expression:
  expression
  {
    $$ = &LogicalExpression{VitessExpression: $1}
  }
| table_name opt_percent
  {
    $$ = &LogicalExpression{CursorPercentName: $1, CursorPercentOpt: $2}
  }

plsql_expression:
  cursor_expression
  {
    $$ = &PlsqlExpression{CursorExpre: $1}
  }
| logical_expression
  {
    $$ = &PlsqlExpression{LogicalExpre: $1}
  }

opt_subtype_range:
  {
    $$ =nil
  }
| RANGE value_expression DOUBLE_POINT value_expression
  {
    $$ = &SubTypeRange{LowValue: $2, HighValue: $4}
  }

opt_is_select:
  {
    $$ = nil
  }
| IS base_select opt_order_by opt_limit opt_lock
  {
    sel := $2.(*Select)
    sel.OrderBy = $3
    sel.Limit = $4
    sel.Lock = $5
    $$ = sel
  }

opt_cursor_return:
  {
    $$ = nil
  }
| RETURN type_spec
  {
    $$ = $2
  }

opt_parameter_spec_list:
  {
    $$ = nil
  }
| '(' parameter_spec_list ')'
  {
    $$ = $2
  }

parameter_spec_list:
  parameter_spec
  {
    $$ = ParameterSpecList{$1}
  }
| parameter_spec_list ',' parameter_spec
  {
    $$ = append($1, $3)
  }

parameter_spec:
  ID opt_parameter_spec_type opt_default_value_part
  {
    $$ = &ParameterSpec{ParameterName: string($1), TypeOpt: $2, DefaultValueOpt: $3}
  }

opt_parameter_spec_type:
  {
    $$ = nil
  }
| type_spec
  {
    $$ = &ParameterSpecType{TypeSpec: $1}
  }
| IN type_spec
  {
    $$ = &ParameterSpecType{ModeOpt: string($1), TypeSpec: $2}
  }

opt_parameter_list:
  {
    $$ = nil
  }
| '(' parameter_list ')'
  {
    $$ = $2
  }

parameter_list:
  parameter
  {
    $$ = ParameterList{$1}
  }
| parameter_list ',' parameter
  {
    $$ = append($1, $3)
  }

parameter:
  ID opt_mode opt_type_spec
  {
    $$ = &Parameter{ParameterName: string($1), ModeOpt: $2, TypeOpt: $3}
  }

opt_type_spec:
  {
    $$ = nil
  }
| type_spec
  {
    $$ = $1
  }

type_spec:
  dataType
  {
    $$ = &TypeSpec{DataType: $1}
  }
| opt_ref type_name opt_type_name
  {
    $$ = &TypeSpec{RefOpt: $1, TypeName: $2, TypeNameOpt: $3}
  }

opt_local:
  {
    $$ = ""
  }
| LOCAL
  {
    $$ = string($1)
  }

opt_year_day:
  YEAR
  {
    $$ = string($1)
  }
| DAY
  {
    $$ = string($1)
  }

opt_month_second:
  MONTH
  {
    $$ = string($1)
  }
| SECOND
  {
    $$ = string($1)
  }

dataType:
  native_datatype_element
  {
    $$ = &DataType{NativeType: $1}
  }
| native_datatype_element WITH opt_local TIME ZONE
  {
    $$ = &DataType{NativeType: $1, OptWith: string($2), OptLocal: $3}
  }
| native_datatype_element CHARACTER SET id_expression_list
  {
    $$ = &DataType{NativeType: $1, OptCharSet: $4}
  }
| INTERVAL opt_year_day '(' plsql_expression ')' TO opt_year_day '(' plsql_expression ')'
  {
    $$ = &DataType{FromTimePeriod: $2, OptIntervalFrom: $4, ToTimePeriod: $7, OptIntervalTo: $9}
  }

opt_type_name:
  {
    $$ = ""
  }
| '%' ROWTYPE
  {
    $$ = "%ROWTYPE"
  }
| '%' TYPE
  {
    $$ = "%TYPE"
  }

opt_ref:
  {
    $$ = ""
  }
| REF
  {
    $$ = string($1)
  }

type_name:
  id_expression
  {
    $$ = $1.String()
  }
| type_name '.' id_expression
  {
    $$ = $1 + "." + $3.String()
  }

opt_mode:
  {
    $$ = ""
  }
| IN
  {
    $$ = string($1)
  }
| OUT
  {
    $$ = string($1)
  }
| IN OUT
  {
    $$ = string($1) + " " + string($2)
  }
| IN OUT NOCOPY
  {
    $$ = string($1) + " " + string($2) + " " + string($3)
  }
| OUT NOCOPY
  {
    $$ = string($1) + " " + string($2)
  }

is_or_as:
  IS
  {
    $$ = string($1)
  }
| AS
  {
    $$ = string($1)
  }

opt_invoker_rights_clause:
  {
    $$ = ""
  }
| AUTHID CURRENT_USER
  {
    $$ = string($1) + " " + string($2)
  }
| AUTHID DEFINER
  {
    $$ = string($1) + " " + string($2)
  }

native_datatype_element:
  NUMBER
  // todo more


id_expression:
  regular_id
  {
    $$ = &IdExpression{RegularIdOpt: $1.String()}
  }
//| delimited_id
//  {
//
//  }
  // todo

regular_id:
  sql_id
  {
    $$ = $1
  }
  // todo complete

opt_name:
  {
    $$ = ""
  }
| ID
  {
    $$ = string($1)
  }

opt_constant:
  {
    $$ = ""
  }
| CONSTANT
  {
    $$ = string($1)
  }

opt_not_null:
  {
    $$ = ""
  }
| NOT NULL
  {
    $$ = string($1) + " " + string($2)
  }

opt_default_value_part:
  {
    $$ = nil
  }
| ASSIGN plsql_expression
  {
    $$ = &DefaultValuePart{AssignExpression: $2}
  }
| DEFAULT plsql_expression
  {
    $$ = &DefaultValuePart{DefaultExpression: $2}
  }
