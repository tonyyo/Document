create database db;
use db
create table t1(id int, name varchar(20));
create table t2(count int);
insert t2 values(1);
insert t1 values(0, 'vincent');
INSERT t1 VALUES(1, 'bob');
select * from t1;

delimiter //
	CREATE FUNCTION func(v_in IN INT)
	RETURN INT
	IS
	DECLARE
		abc INT;
	BEGIN
		IF v_in > 0 THEN
			RETURN 1;
		ELSIF v_in < 0 THEN
			RETURN -1;
		ELSIF v_in = 0 THEN
			RETURN 0;
		ELSE
			RETURN 100;
		END IF;
	END;
//
delimiter ;
