


DELIMITER $$

DROP PROCEDURE IF EXISTS pay$$
CREATE PROCEDURE pay(`v_w_id` int(11), `v_c_w_id` int(11), `v_h_amount` decimal(12,2), `v_d_id` int(11), `v_c_d_id` int(11), `v_c_id` int(11), `v_c_last` varchar(16), `v_c_by_name` bool)
BEGIN
  DECLARE v_w_street_1 varchar(20);
  DECLARE v_w_street_2 varchar(20);
  DECLARE v_w_city varchar(20);
  DECLARE v_w_state char(2);
  DECLARE v_w_zip char(9);
  DECLARE v_w_name varchar(10);
  DECLARE v_d_street_1 varchar(20);
  DECLARE v_d_street_2 varchar(20);
  DECLARE v_d_city varchar(20);
  DECLARE v_d_state char(2);
  DECLARE v_d_zip char(9);
  DECLARE v_d_name varchar(10);

  DECLARE v_c_id int(11);
  DECLARE v_c_first varchar(16);
  DECLARE v_c_middle char(2);
  DECLARE v_c_street_1 varchar(20);
  DECLARE v_c_street_2 varchar(20);
  DECLARE v_c_city varchar(20);
  DECLARE v_c_state char(2);
  DECLARE v_c_zip char(9);
  DECLARE v_c_phone char(16);
  DECLARE v_c_credit char(2);
  DECLARE v_c_credit_lim decimal(12,2);
  DECLARE v_c_discount decimal(4,4);
  DECLARE v_c_balance decimal(12,2);
  DECLARE v_c_since timestamp;
  DECLARE v_c_data varchar(500);

  DECLARE v_namecnt int(11);
  DECLARE v_c_new_data varchar(500);

  DECLARE v_i INT DEFAULT 1;

  DECLARE payCursorCustByName CURSOR FOR SELECT c_first, c_middle, c_id, c_street_1, c_street_2, c_city, c_state, c_zip, c_phone, c_credit, c_credit_lim, c_discount, c_balance, c_since
          FROM customer WHERE c_w_id = v_c_w_id AND c_d_id = v_c_d_id AND c_last = v_c_last
          ORDER BY c_w_id, c_d_id, c_last, c_first ;


  IF EXISTS (SELECT * FROM warehouse WHERE w_id = v_w_id) THEN
    UPDATE warehouse SET w_ytd = w_ytd + v_h_amount  WHERE w_id = v_w_id;

    SELECT w_street_1, w_street_2, w_city, w_state, w_zip, w_name
        INTO v_w_street_1, v_w_street_2, v_w_city, v_w_state, v_w_zip, v_w_name
        FROM warehouse WHERE w_id = v_w_id;

    UPDATE district SET d_ytd = d_ytd + v_h_amount WHERE d_w_id = v_w_id AND d_id = v_d_id;

    SELECT d_street_1, d_street_2, d_city, d_state, d_zip, d_name
        INTO v_d_street_1, v_d_street_2, v_d_city, v_d_state, v_d_zip, v_d_name
        FROM district WHERE d_w_id = v_w_id AND d_id = v_d_id;

  END IF;

  IF EXISTS (SELECT * FROM customer WHERE c_w_id = v_c_w_id) THEN
    IF v_c_by_name THEN
      SELECT count(*) AS namecnt 
          INTO v_namecnt
          FROM customer
          WHERE c_last = v_c_last  AND c_d_id = v_c_d_id AND c_w_id = v_c_w_id;

      OPEN payCursorCustByName;

      IF v_namecnt % 2 = 1 THEN
        SET v_namecnt = v_namecnt + 1;
      END IF;

      SET v_i = 1;
      REPEAT
        FETCH payCursorCustByName INTO v_c_first, v_c_middle, v_c_id, v_c_street_1, v_c_street_2, v_c_city, v_c_state, v_c_zip, v_c_phone, v_c_credit, v_c_credit_lim, v_c_discount, v_c_balance, v_c_since;

        SET v_i = v_i + 1;
      UNTIL v_i > v_namecnt DIV 2
      END REPEAT;

      CLOSE payCursorCustByName;

    ELSE
      SELECT c_first, c_middle, c_last, c_street_1, c_street_2, c_city, c_state, c_zip, c_phone, c_credit, c_credit_lim, c_discount, c_balance, c_since
      INTO v_c_first, v_c_middle, v_c_last, v_c_street_1, v_c_street_2, v_c_city, v_c_state, v_c_zip, v_c_phone, v_c_credit, v_c_credit_lim, v_c_discount, v_c_balance, v_c_since
          FROM customer WHERE c_w_id = v_c_w_id AND c_d_id = v_c_d_id AND c_id = v_c_id;

    END IF;

    SET v_c_balance = v_c_balance + v_h_amount;

    IF v_c_credit = 'BC' THEN
      SELECT c_data
          INTO v_c_data
          FROM customer WHERE c_w_id = v_c_w_id AND c_d_id = v_c_d_id AND c_id = v_c_id;

      SET v_c_new_data = CONCAT(v_c_id, ' ', v_c_d_id, ' ', v_c_w_id, ' ', v_d_id, ' ', v_w_id, ' ', v_h_amount, '|');

      IF LENGTH(v_c_data) > LENGTH(v_c_new_data) THEN
        SET v_c_new_data = CONCAT(v_c_new_data, SUBSTRING(v_c_data, 1, LENGTH(v_c_data)-LENGTH(v_c_new_data)));
      ELSE
        SET v_c_new_data = CONCAT(v_c_new_data, SUBSTRING(v_c_data, 1, 500-LENGTH(v_c_new_data)));
      END IF;

      UPDATE customer SET c_balance = v_c_balance, c_data = v_c_new_data
          WHERE c_w_id = v_c_w_id AND c_d_id = v_c_d_id AND c_id = v_c_id;

    ELSE

      UPDATE customer SET c_balance = v_c_balance
          WHERE c_w_id = v_c_w_id AND c_d_id = v_c_d_id AND c_id = v_c_id;

    END IF;

  END IF;
    
END$$

DELIMITER ;









----------

--- w_id, c_w_id, h_amount, d_id, c_d_id, c_id, c_last,        c_by_name

BEGIN;
select * from warehouse where w_id=9;
select * from district where d_w_id=9 and d_id=1;
CALL pay(9, 11,   0.01,     1,    2,      3,    'ABLEABLEABLE', 0);
select * from warehouse where w_id=9;
select * from district where d_w_id=9 and d_id=1;
CALL pay(9, 11,   0.01,     1,    2,      3,    'ABLEABLEABLE', 1);
select * from warehouse where w_id=9;
select * from district where d_w_id=9 and d_id=1;
rollback;
select * from warehouse where w_id=9;
select * from district where d_w_id=9 and d_id=1;







-----------


mfed version


DELIMITER $$

DROP PROCEDURE IF EXISTS payment$$
CREATE PROCEDURE payment(`v_w_id` int(11), `v_c_w_id` int(11), `v_h_amount` decimal(12,2), `v_d_id` int(11), `v_c_d_id` int(11), `v_c_id` int(11), `v_c_last` varchar(16), `v_c_by_name` bool)
BEGIN
  DECLARE v_w_street_1 varchar(20);
  DECLARE v_w_street_2 varchar(20);
  DECLARE v_w_city varchar(20);
  DECLARE v_w_state char(2);
  DECLARE v_w_zip char(9);
  DECLARE v_w_name varchar(10);
  DECLARE v_d_street_1 varchar(20);
  DECLARE v_d_street_2 varchar(20);
  DECLARE v_d_city varchar(20);
  DECLARE v_d_state char(2);
  DECLARE v_d_zip char(9);
  DECLARE v_d_name varchar(10);

  DECLARE v_c_id int(11);
  DECLARE v_c_first varchar(16);
  DECLARE v_c_middle char(2);
  DECLARE v_c_street_1 varchar(20);
  DECLARE v_c_street_2 varchar(20);
  DECLARE v_c_city varchar(20);
  DECLARE v_c_state char(2);
  DECLARE v_c_zip char(9);
  DECLARE v_c_phone char(16);
  DECLARE v_c_credit char(2);
  DECLARE v_c_credit_lim decimal(12,2);
  DECLARE v_c_discount decimal(4,4);
  DECLARE v_c_balance decimal(12,2);
  DECLARE v_c_since timestamp;
  DECLARE v_c_data varchar(500);

  DECLARE v_h_data varchar(24);

  DECLARE v_namecnt int(11);
  DECLARE v_c_new_data varchar(500);

  DECLARE v_i INT DEFAULT 1;
  DECLARE v_done INT DEFAULT 0;

  DECLARE payCursorCustByName CURSOR FOR SELECT c_first, c_middle, c_id, c_street_1, c_street_2, c_city, c_state, c_zip, c_phone, c_credit, c_credit_lim, c_discount, c_balance, c_since
          FROM customer WHERE c_w_id = v_c_w_id AND c_d_id = v_c_d_id AND c_last = v_c_last
          ORDER BY c_w_id, c_d_id, c_last, c_first ;
  DECLARE CONTINUE HANDLER FOR NOT FOUND SET v_done = 1;



    UPDATE warehouse SET w_ytd = w_ytd + v_h_amount  WHERE w_id = v_w_id;

    SELECT w_street_1, w_street_2, w_city, w_state, w_zip, w_name
        INTO v_w_street_1, v_w_street_2, v_w_city, v_w_state, v_w_zip, v_w_name
        FROM warehouse WHERE w_id = v_w_id;

    UPDATE district SET d_ytd = d_ytd + v_h_amount WHERE d_w_id = v_w_id AND d_id = v_d_id;

    SELECT d_street_1, d_street_2, d_city, d_state, d_zip, d_name
        INTO v_d_street_1, v_d_street_2, v_d_city, v_d_state, v_d_zip, v_d_name
        FROM district WHERE d_w_id = v_w_id AND d_id = v_d_id;


    IF v_c_by_name THEN
      SELECT count(*) AS namecnt 
          INTO v_namecnt
          FROM customer
          WHERE c_last = v_c_last  AND c_d_id = v_c_d_id AND c_w_id = v_c_w_id;

      OPEN payCursorCustByName;

      IF v_namecnt % 2 = 1 THEN
        SET v_namecnt = v_namecnt + 1;
      END IF;

      SET v_i = 1;
      REPEAT
        FETCH payCursorCustByName INTO v_c_first, v_c_middle, v_c_id, v_c_street_1, v_c_street_2, v_c_city, v_c_state, v_c_zip, v_c_phone, v_c_credit, v_c_credit_lim, v_c_discount, v_c_balance, v_c_since;
        SET v_i = v_i + 1;
      UNTIL v_done = 1 OR v_i > v_namecnt DIV 2
      END REPEAT;

      CLOSE payCursorCustByName;

    ELSE
      SELECT c_first, c_middle, c_last, c_street_1, c_street_2, c_city, c_state, c_zip, c_phone, c_credit, c_credit_lim, c_discount, c_balance, c_since
      INTO v_c_first, v_c_middle, v_c_last, v_c_street_1, v_c_street_2, v_c_city, v_c_state, v_c_zip, v_c_phone, v_c_credit, v_c_credit_lim, v_c_discount, v_c_balance, v_c_since
          FROM customer WHERE c_w_id = v_c_w_id AND c_d_id = v_c_d_id AND c_id = v_c_id;

    END IF;

    SET v_c_balance = v_c_balance + v_h_amount;

    IF v_c_credit = 'BC' THEN
      SELECT c_data
          INTO v_c_data
          FROM customer WHERE c_w_id = v_c_w_id AND c_d_id = v_c_d_id AND c_id = v_c_id;

      SET v_c_new_data = CONCAT(v_c_id, ' ', v_c_d_id, ' ', v_c_w_id, ' ', v_d_id, ' ', v_w_id, ' ', v_h_amount, '|');

      IF LENGTH(v_c_data) > LENGTH(v_c_new_data) THEN
        SET v_c_new_data = CONCAT(v_c_new_data, SUBSTRING(v_c_data, 1, LENGTH(v_c_data)-LENGTH(v_c_new_data)));
      ELSE
        SET v_c_new_data = CONCAT(v_c_new_data, SUBSTRING(v_c_data, 1, 500-LENGTH(v_c_new_data)));
      END IF;

      UPDATE customer SET c_balance = v_c_balance, c_data = v_c_new_data
          WHERE c_w_id = v_c_w_id AND c_d_id = v_c_d_id AND c_id = v_c_id;

    ELSE

      UPDATE customer SET c_balance = v_c_balance
          WHERE c_w_id = v_c_w_id AND c_d_id = v_c_d_id AND c_id = v_c_id;

    END IF;

    IF LENGTH(v_w_name) > 10 THEN
      SET v_w_name = SUBSTRING(v_w_name, 1, 10);
    END IF;  

    IF LENGTH(v_d_name) > 10 THEN
      SET v_d_name = SUBSTRING(v_d_name, 1, 10);
    END IF;

    SET v_h_data = CONCAT(v_w_name, '    ', v_d_name);

    INSERT INTO history (h_c_d_id, h_c_w_id, h_c_id, h_d_id, h_w_id, h_date, h_amount, h_data)
        SELECT v_c_d_id,v_c_w_id,v_c_id,v_d_id,v_w_id,CURRENT_TIMESTAMP(),v_h_amount,v_h_data FROM dual;
    
END$$

DELIMITER ;



----- w_id, c_w_id, h_amount, d_id, c_d_id, c_id, c_last,        c_by_name

BEGIN;
select * from warehouse where w_id=9;
select * from district where d_w_id=9 and d_id=1;
CALL payment(9, 11,   0.01,     1,    2,      3,    'ABLEABLEABLE', 0);
select * from warehouse where w_id=9;
select * from district where d_w_id=9 and d_id=1;
CALL payment(9, 11,   0.01,     1,    2,      3,    'ABLEABLEABLE', 1);
select * from warehouse where w_id=9;
select * from district where d_w_id=9 and d_id=1;
rollback;
select * from warehouse where w_id=9;
select * from district where d_w_id=9 and d_id=1;











