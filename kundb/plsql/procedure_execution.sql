create database db;
use db
create table t1(id int, name varchar(20));
create table t2(count int);
insert t2 values(1);
insert t1 values(0, 'vincent');
INSERT t1 VALUES(1, 'bob');
select * from t1;

delimiter //
         CREATE PROCEDURE pro1(v_in IN INT) 
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

call pro1(0);

