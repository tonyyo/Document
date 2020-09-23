
DELIMITER $$

DROP PROCEDURE IF EXISTS pay$$
CREATE PROCEDURE oracle_pay11(`v_w_id` int(11), `v_c_w_id` int(11), `v_h_amount` decimal(12,2), `v_d_id` int(11), `v_c_d_id` int(11), `v_c_id` int(11), `v_c_last` varchar(16), `v_c_by_name` int(11))
IS
DECLARE
  v_w_street_1 varchar(20);
  v_w_street_2 varchar(20);
  v_w_city varchar(20);
  v_w_state char(2);
  v_w_zip char(9);
  v_w_name varchar(10);
  v_d_street_1 varchar(20);
  v_d_street_2 varchar(20);
  v_d_city varchar(20);
  v_d_state char(2);
  v_d_zip char(9);
  v_d_name varchar(10);

  v_c_first varchar(16);
  v_c_middle char(2);
  v_c_street_1 varchar(20);
  v_c_street_2 varchar(20);
  v_c_city varchar(20);
  v_c_state char(2);
  v_c_zip char(9);
  v_c_phone char(16);
  v_c_credit char(2) := 'AC';
  v_c_credit_lim decimal(12,2);
  v_c_discount decimal(4,4);
  v_c_balance decimal(12,2);
  v_c_since timestamp;
  v_c_data varchar(500);
 
  v_namecnt int(11);
  v_c_new_data varchar(500);

  v_i INT DEFAULT 1;
  
  c1 INT;
  c2 INT;  
  
BEGIN

  SELECT count(*) INTO c1 FROM bmsql_warehouse WHERE w_id = v_w_id;

  IF c1 > 0 THEN
    UPDATE bmsql_warehouse SET w_ytd = 0.01  WHERE w_id = 9;

    SELECT w_street_1, w_street_2, w_city, w_state, w_zip, w_name
        INTO v_w_street_1, v_w_street_2, v_w_city, v_w_state, v_w_zip, v_w_name
        FROM bmsql_warehouse WHERE w_id = v_w_id;

    UPDATE bmsql_district SET d_ytd = 0.01 WHERE d_w_id = 9 AND d_id = 1;

    SELECT d_street_1, d_street_2, d_city, d_state, d_zip, d_name
        INTO v_d_street_1, v_d_street_2, v_d_city, v_d_state, v_d_zip, v_d_name
        FROM bmsql_district WHERE d_w_id = v_w_id AND d_id = v_d_id;

  END IF;
    
  SELECT count(*) INTO c2 FROM bmsql_customer WHERE c_w_id = v_c_w_id;

  IF c2 > 0 THEN
    IF v_c_by_name = 1 THEN
      SELECT count(*) AS namecnt 
          INTO v_namecnt
          FROM bmsql_customer
          WHERE c_last = v_c_last  AND c_d_id = v_c_d_id AND c_w_id = v_c_w_id;

      IF v_namecnt % 2 = 1 THEN
        v_namecnt := v_namecnt + 1;
      END IF;

      v_i := 1;
      LOOP
        SELECT c_first, c_middle, c_id, c_street_1, c_street_2, c_city, c_state, c_zip, c_phone, c_credit, c_credit_lim, c_discount, c_balance, c_since 
        INTO v_c_first, v_c_middle, v_c_id, v_c_street_1, v_c_street_2, v_c_city, v_c_state, v_c_zip, v_c_phone, v_c_credit, v_c_credit_lim, v_c_discount, v_c_balance, v_c_since           
        FROM bmsql_customer 
        WHERE c_w_id = v_c_w_id AND c_d_id = v_c_d_id AND c_last = v_c_last
        ORDER BY c_w_id, c_d_id, c_last, c_first ;

        v_i := v_i + 1;
      	EXIT WHEN v_i > v_namecnt DIV 2;
      END LOOP;

    ELSE
      SELECT c_first, c_middle, c_last, c_street_1, c_street_2, c_city, c_state, c_zip, c_phone, c_credit, c_credit_lim, c_discount, c_balance, c_since
      INTO v_c_first, v_c_middle, v_c_last, v_c_street_1, v_c_street_2, v_c_city, v_c_state, v_c_zip, v_c_phone, v_c_credit, v_c_credit_lim, v_c_discount, v_c_balance, v_c_since
      FROM bmsql_customer 
      WHERE c_w_id = v_c_w_id AND c_d_id = v_c_d_id AND c_id = v_c_id;

    END IF; 

    v_c_balance := v_h_amount;

    IF v_c_credit = 'BC' THEN
      SELECT c_data
          INTO v_c_data
          FROM bmsql_customer WHERE c_w_id = v_c_w_id AND c_d_id = v_c_d_id AND c_id = v_c_id;

      UPDATE bmsql_customer SET c_balance = 0.01, c_data = v_c_new_data
          WHERE c_w_id = 11 AND c_d_id = 2 AND c_id = 3;

    ELSE

      UPDATE bmsql_customer SET c_balance = 0.01
          WHERE c_w_id = 11 AND c_d_id = 2 AND c_id = 3;

    END IF;

  END IF;
    
END;
$$

DELIMITER ;


