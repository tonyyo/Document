create database db;
use db
create table t1(id int, name varchar(20));
create table t2(count int);
insert t2 values(1);
insert t1 values(0, 'vincent');
INSERT t1 VALUES(1, 'bob');
select * from t1;

delimiter //
 CREATE PROCEDURE proc2(v_in IN INT) 
              IS
              DECLARE
                v0 INT := 0;
		        v1 INT := 1;
		        v2 INT := 1;
		        v3 INT := 1;
		        upper INT;
              BEGIN 
                 SELECT count INTO upper FROM t2 order by count limit 1;
                 WHILE v0 < upper
                 LOOP
                    INSERT t2 VALUES(2);
                    v0 := v0 + 1;
                 END LOOP;
                 
                 UPDATE t1 SET name = 'mary' WHERE id = 0;
                 
                 DELETE FROM t1 WHERE NAME = 'bob';
                 
                 IF v_in > 0 THEN
                   INSERT t1 VALUES (1, 'positive');
                 ELSIF v_in < 0 THEN
                   INSERT t1 VALUES (-1, 'negative');
                 ELSE
                   INSERT t1 VALUES (0, 'zero');
                 END IF;
                 
                 WHILE v1 < 10
                 LOOP
                   v1 := v1 + 10;
                   INSERT t1 VALUES (1, 'while');
                 END LOOP;
                 
                 LOOP
                   v2 := v2 + 10;
                   INSERT t1 VALUES (2, 'basicLoop');
                   IF v2 > 10 THEN
                     EXIT;
                   END IF;
                 END LOOP;
                                    
                 <<outer_loop>>  
                 FOR i IN REVERSE 1 .. 2 
                 LOOP
                    <<inner_loop>>
                    FOR j IN 1 .. 2
                    LOOP
                       v3 := v3 + 1;
                       INSERT t1 VALUES (3, 'forLoop');
                       EXIT outer_loop WHEN v3 > 5;
                    END LOOP inner_loop;
                    v3 := v3 + 5;	
                 END LOOP outer_loop;                                  
            END;
//            
delimiter ;

call proc1(0);

我稍微看了下 plsql_vschema.go 里的代码，对应expression这一块，每个expression都需要大概10多行代码去序列化反序列化，之后function_expressions.go 里expression越来越多。
这一块序/反序的逻辑应该是.可以找一种方法是高度可重用的，应该能用比较少的代码就可以实现了。比如
```
message BinaryExpr {
  typ = Addition or Bitwise ComparisonLargerThan ComparisonEqual ... // 伪代码表示二元表达式类型
}
```
然后在`updateBinaryExprVSchema`函数里，就可以用
```
case *evalengine.Bitwise:
  typ = Bitwise
case *evalengine.Comparison:
  swtich e.Operator {
    case sqlparser.GreaterThanStr
      typ = ComparisonLargerThan
    ...
  }
...
return &vschemapb.BinaryExpr{typ}
```
包括可以设计一个evalengine.Expr -> typ 的map，这样连swtich case也可能省很多


这个map可能有点理想化，毕竟像evalengine.Comparison到vschemapb.Comparison在这种思路下是一对多的转换，根据Operator的不同转换成 vschemapb.ComparisonGreatorThan， vschemapb.ComparisonEqual等。如果要使用map，则标准化一个expr的id，evalengine下的每个表达式有一个GetId的方法，代表不同类型的表达式，而evalengine.Comparison根据Operator的不同返回不同的id。这样 typ = exprIdMap[evalengine.Expr.GetId()] 这样直接一行就好了

像 UnaryOp，BinaryOp 还是保持现在的逻辑，因为每个逻辑都不一样

还有在反序列化时，可以考虑利用反射来初始化evalengine.Expr  比如保留一个map 从typ -> evalengine.Expr实例，然后通过 mapEvalExpr[vschemapb.Expr.type]得到实例instance，然后 reflect.New(reflect.TypeOf(instance))才创建新的expr。对于像Comparison这种有参数的Expr还是需要特殊处理。这一步也可以省代码而且逻辑能进一步抽象，后续添加新的evalengine expr实现的时候，理想情况下需要在mapEvalExpr和exprIdMap 两个map里面加两行代码就行

命名随便命的  不要参考。如果这样实现了  上述提到的 `typ` 就是expression的`实现ID` 不同的ID代表不同的实现逻辑，ComparisonGreatorThan，.ComparisonEqual，Addition，Belong，BelongNot等


