CREATE OR REPLACE PROCEDURE NewOrder
(
    IN_mW_ID                IN   INTEGER,       
    IN_mD_ID                IN   INTEGER,       
    IN_mC_ID                IN   INTEGER,        
    IN_mOL_CNT              IN   INTEGER,      
    IN_mALL_LOCAL           IN   INTEGER,
    IN_ItemString           IN   VARCHAR(2000),
    OUT_mC_LAST            OUT   VARCHAR(16),
    OUT_mC_CREDIT          OUT   VARCHAR(2),
    OUT_mC_DISCOUNT        OUT   NUMERIC(4, 4),
    OUT_mW_TAX             OUT   NUMERIC(4, 4),
    OUT_mD_TAX             OUT   NUMERIC(4, 4),
    OUT_mO_ID              OUT   NATIVE_INTEGER,
    OUT_mO_ENTRY_D         OUT   VARCHAR(21),
    OUT_mTotalAmount       OUT   NUMERIC(15, 2),
    OUT_mItemString        OUT   VARCHAR(2000),
    OUT_mISROLLBACK        OUT   NATIVE_INTEGER,
    OUT_mSuccess           OUT   NATIVE_INTEGER,
    OUT_mMessage           OUT   VARCHAR(2000)
)
IS
    total                        NUMERIC(15, 2);
    sNativeError                 INTEGER;

    sIsRollbacked                CHAR(1);
    sItemString                  VARCHAR(2000);
    NO_w_id                      INTEGER;
    NO_d_id                      INTEGER;
    NO_c_id                      INTEGER;
    NO_o_ol_cnt                  INTEGER;
    NO_o_all_local               NUMERIC(1);
    NO_c_discount                NUMERIC(4, 4);
    NO_c_last                    VARCHAR(16);
    NO_c_credit                  CHAR(2);
    NO_w_tax                     NUMERIC(4, 4);

    NO_d_tax                     NUMERIC(4, 4);
    NO_o_entry_d                 VARCHAR(21);
    NO_o_id                      INTEGER;
    NO_i_name                    VARCHAR(25);
    NO_i_price                   NUMERIC(5, 2);
    NO_i_data                    VARCHAR(50);
    NO_ol_i_id                   INTEGER;
    NO_s_quantity                NUMERIC(4);
    NO_org_quantity              NUMERIC(4);

    NO_s_data                    VARCHAR(50);
    NO_ol_dist_info              CHAR(24);
    NO_ol_supply_w_id            INTEGER;
    NO_ol_amount                 NUMERIC(6, 2);
    NO_ol_number                 INTEGER;
    NO_ol_quantity               NUMERIC(2);
    NO_s_remote_cnt_increment    NUMERIC(8);
    NO_s_BG                      CHAR(1);

    sTotalAmount                 NUMERIC(15, 2);

    sTmpTxt                      VARCHAR(4000);

    PROCEDURE SETNODATA
    (
        aMessage IN VARCHAR(2000)
    )
    IS
    BEGIN
        sNativeError := SQLCODE;
        OUT_mMessage := aMessage;
    END;

    PROCEDURE SETERR
    IS
    BEGIN
        sNativeError := SQLCODE;
        OUT_mMessage := SQLERRM;
    END;
BEGIN

    ----------------------------------------------
    -- Assign Input Param, Initialize PSM Variable
    ----------------------------------------------

    NO_w_id        := IN_mW_ID;
    NO_d_id        := IN_mD_ID;
    NO_c_id        := IN_mC_ID;
    NO_o_ol_cnt    := IN_mOL_CNT;
    NO_o_all_local := IN_mALL_LOCAL;
    NO_o_entry_d   := TO_CHAR(sysdate, 'yyyy-mm-dd hh24:mi:ss');
    sItemString    := IN_ItemString;

    << RAMP_RETRY >>

    sIsRollbacked  := 0;
    total          := 0;
    sTmpTxt        := '';
    sNativeError   := 0;

    BEGIN
        SELECT /*+ USE_NL(WAREHOUSE, CUSTOMER) INDEX( WAREHOUSE, WAREHOUSE_PK_IDX ) 
                   INDEX( CUSTOMER, CUSTOMER_PK_IDX ) */
               c_discount, c_last, c_credit, w_tax 
          INTO NO_c_discount,
               NO_c_last,
               NO_c_credit,
               NO_w_tax
          FROM warehouse, customer 
         WHERE w_id   = NO_w_id AND 
               c_w_id = w_id AND 
               c_d_id = NO_d_id AND 
               c_id   = NO_c_id ;
    EXCEPTION WHEN NO_DATA_FOUND THEN SETNODATA('NO DATA FOUND. INVALID W_ID or D_ID or C_ID.');goto SQL_FINISH;
              WHEN OTHERS        THEN SETERR;goto SQL_FINISH;
    END;


    BEGIN
        UPDATE district 
           SET d_next_o_id = d_next_o_id + 1 
         WHERE d_id   = NO_d_id AND 
               d_w_id = NO_w_id
        RETURNING OLD d_next_o_id, d_tax
        INTO NO_o_id, NO_d_tax;
    EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
    END;
 

    BEGIN
        INSERT INTO ORDERS (o_id, o_d_id, o_w_id, o_c_id, 
                            o_entry_d, o_ol_cnt, o_all_local) 
        VALUES (NO_o_id, NO_d_id, NO_w_id, NO_c_id, TO_DATE(NO_o_entry_d,'yyyy-mm-dd hh24:mi:ss'), NO_o_ol_cnt, NO_o_all_local);
    EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
    END;


    BEGIN
        INSERT INTO NEW_ORDER (no_o_id, no_d_id, no_w_id) 
        VALUES (NO_o_id, NO_d_id, NO_w_id);
    EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
    END;


    FOR NO_ol_number IN 1 .. NO_o_ol_cnt
    LOOP
        NO_ol_i_id        := SPLIT_PART( sItemString, '|', (NO_ol_number - 1) * 3 + 1);
        NO_ol_supply_w_id := SPLIT_PART( sItemString, '|', (NO_ol_number - 1) * 3 + 2);
        NO_ol_quantity    := SPLIT_PART( sItemString, '|', (NO_ol_number - 1) * 3 + 3);

        BEGIN
            SELECT 
                   i_price, NVL(i_name, ' '), i_data 
              INTO NO_i_price,
                   NO_i_name,
                   NO_i_data
              FROM item 
             WHERE i_id = NO_ol_i_id;
        EXCEPTION WHEN NO_DATA_FOUND THEN sIsRollbacked := 1; CONTINUE;
                  WHEN OTHERS THEN SETERR;goto SQL_FINISH;
        END;

        
        BEGIN
            SELECT 
                      s_data, 
                      s_quantity,
                      CASE NO_d_id
                          WHEN 1 THEN  s_dist_01
                          WHEN 2 THEN  s_dist_02
                          WHEN 3 THEN  s_dist_03
                          WHEN 4 THEN  s_dist_04
                          WHEN 5 THEN  s_dist_05
                          WHEN 6 THEN  s_dist_06
                          WHEN 7 THEN  s_dist_07
                          WHEN 8 THEN  s_dist_08
                          WHEN 9 THEN  s_dist_09
                          WHEN 10 THEN  s_dist_10
                          ELSE NULL
                      END,
                      CASE 
                          WHEN ((POSITION( 'ORIGINAL' IN NO_i_data) <> 0) AND (POSITION( 'ORIGINAL' IN s_data) <> 0 ) )
                          THEN 'B' ELSE 'G'
                      END,
                      CASE WHEN (s_quantity - NO_ol_quantity >= 10  )
                           THEN (s_quantity - NO_ol_quantity)
                           ELSE s_quantity - NO_ol_quantity + 91
                      END,
                      CASE WHEN (NO_ol_supply_w_id = NO_w_id ) THEN 0 ELSE 1 END,
                      (NO_ol_quantity * NO_i_price),
                      (total + (NO_ol_quantity * NO_i_price) )
             INTO 
                    NO_s_data,
                    NO_org_quantity,
                    NO_ol_dist_info,
                    NO_s_BG,
                    NO_s_quantity,
                    NO_s_remote_cnt_increment,
                    NO_ol_amount,
                    total
             FROM stock 
            WHERE s_i_id = NO_ol_i_id AND 
                  s_w_id = NO_ol_supply_w_id 
              FOR UPDATE;
        EXCEPTION WHEN NO_DATA_FOUND THEN SETNODATA('NO DATA FOUND. INVALID S_W_ID.');goto SQL_FINISH;
                  WHEN OTHERS        THEN SETERR;goto SQL_FINISH;
        END;
     

        BEGIN
            UPDATE stock 
               SET s_quantity   = NO_s_quantity, 
                   s_ytd        = s_ytd + NO_ol_quantity, 
                   s_order_cnt  = s_order_cnt + 1, 
                   s_remote_cnt = s_remote_cnt + NO_s_remote_cnt_increment
             WHERE s_i_id = NO_ol_i_id AND 
                   s_w_id = NO_ol_supply_w_id ;
        EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
        END;


        BEGIN
            INSERT INTO order_line (ol_o_id, ol_d_id, ol_w_id, 
                                    ol_number, ol_i_id, 
                                    ol_supply_w_id, ol_quantity, 
                                    ol_amount, ol_dist_info) 
            VALUES (NO_o_id, NO_d_id, NO_w_id,
                    NO_ol_number, NO_ol_i_id,
                    NO_ol_supply_w_id, NO_ol_quantity,
                    NO_ol_amount, NO_ol_dist_info) ;
        EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
        END;

        ----------------------------------------------------------
        -- Make Output-Array
        ----------------------------------------------------------
        sTmpTxt := sTmpTxt         || 
                   NO_i_name       || '|' || 
                   NO_org_quantity || '|' ||
                   NO_s_BG         || '|' ||
                   NO_i_price      || '|' || 
                   NO_ol_amount    || '|';

    END LOOP;

    ----------------------------------------------------------
    -- Get Total
     ----------------------------------------------------------
    sTotalAmount := total * (1 + NO_w_tax + NO_d_tax) * (1 - NO_c_discount);

    ----------------------------------------------------------
    -- Output Param
     ----------------------------------------------------------
    OUT_mC_LAST      := NO_c_last;
    OUT_mC_CREDIT    := NO_c_credit;
    OUT_mC_DISCOUNT  := NO_c_discount;
    OUT_mW_TAX       := NO_w_tax;
    OUT_mD_TAX       := NO_d_tax;
    OUT_mO_ID        := NO_o_id;
    OUT_mO_ENTRY_D   := NO_o_entry_d;
    OUT_mTotalAmount := sTotalAmount;
    OUT_mItemString  := sTmpTxt;
    OUT_mISROLLBACK  := sIsRollbacked;
    OUT_mSuccess     := 1;

    IF ( sIsRollbacked = 0 )
    THEN
        COMMIT;
    ELSE
        ROLLBACK;
    END IF;

    goto END_STEP;
    
    << SQL_FINISH >>

    ROLLBACK;

    IF ( (sNativeError = -14007 ) OR (sNativeError = -14032 ) )
    THEN
        goto RAMP_RETRY;
    END IF;

    OUT_mSuccess := 0;

    << END_STEP >>
    NULL;
END;
/

COMMIT;

CREATE OR REPLACE PROCEDURE Payment
(
    IN_mW_ID             IN INTEGER,
    IN_mD_ID             IN INTEGER,
    IN_mC_ID             IN INTEGER,
    IN_mC_W_ID           IN INTEGER,
    IN_mC_D_ID           IN INTEGER,
    IN_mC_LAST           IN VARCHAR(16),
    IN_mH_AMOUNT         IN NUMERIC(6, 2),
    OUT_mW_STREET_1     OUT VARCHAR(20),
    OUT_mW_STREET_2     OUT VARCHAR(20),
    OUT_mW_CITY         OUT VARCHAR(20),
    OUT_mW_STATE        OUT CHAR(2),
    OUT_mW_ZIP          OUT CHAR(9),
    OUT_mH_DATE         OUT VARCHAR(21),
    OUT_mD_STREET_1     OUT VARCHAR(20),
    OUT_mD_STREET_2     OUT VARCHAR(20),
    OUT_mD_CITY         OUT VARCHAR(20),
    OUT_mD_STATE        OUT CHAR(2),
    OUT_mD_ZIP          OUT CHAR(9),
    OUT_mC_ID           OUT NATIVE_INTEGER,
    OUT_mC_FIRST        OUT VARCHAR(16),
    OUT_mC_MIDDLE       OUT CHAR(2),
    OUT_mC_LAST         OUT VARCHAR(16),
    OUT_mC_STREET_1     OUT VARCHAR(20),
    OUT_mC_STREET_2     OUT VARCHAR(20),
    OUT_mC_CITY         OUT VARCHAR(20),
    OUT_mC_STATE        OUT CHAR(2),
    OUT_mC_ZIP          OUT CHAR(9),
    OUT_mC_PHONE        OUT CHAR(16),
    OUT_mC_CREDIT       OUT CHAR(2),
    OUT_mC_CREDIT_LIM   OUT NUMERIC(12, 2),
    OUT_mC_DISCOUNT     OUT NUMERIC(4, 4),
    OUT_mC_BALANCE      OUT NUMERIC(15, 2),
    OUT_mC_SINCE        OUT VARCHAR(21),
    OUT_mC_DATA         OUT VARCHAR(200),
    OUT_mSuccess        OUT NATIVE_INTEGER,
    OUT_mMessage        OUT VARCHAR(2000)
)
IS
    sNativeError      INTEGER;

    PM_w_id           INTEGER;
    PM_d_id           INTEGER;
    PM_c_id           INTEGER;
    PM_w_name         VARCHAR(10);
    PM_w_street_1     VARCHAR(20);
    PM_w_street_2     VARCHAR(20);
    PM_w_city         VARCHAR(20);
    PM_w_state        CHAR(2);
    PM_w_zip          CHAR(9);
    PM_c_d_id         INTEGER;
    PM_c_w_id         INTEGER;
    PM_c_first        VARCHAR(16);
    PM_c_middle       CHAR(2);
    PM_c_last         VARCHAR(16);
    PM_c_street_1     VARCHAR(20);
    PM_c_street_2     VARCHAR(20);
    PM_c_city         VARCHAR(20);
    PM_c_state        CHAR(2);
    PM_c_zip          CHAR(9);
    PM_c_phone        CHAR(16);
    PM_c_since        VARCHAR(21);
    PM_c_credit       CHAR(2);
    PM_c_credit_lim   NUMERIC(12, 2);
    PM_c_discount     NUMERIC(4, 4);
    PM_c_balance      NUMERIC(15, 2);
    PM_c_data         VARCHAR(200);
    PM_h_date         VARCHAR(21);
    PM_h_amount       NUMERIC(6, 2);
    PM_d_name         VARCHAR(10);
    PM_d_street_1     VARCHAR(20);
    PM_d_street_2     VARCHAR(20);
    PM_d_city         VARCHAR(20);
    PM_d_state        CHAR(2);
    PM_d_zip          CHAR(9);
    PM_namecnt        INTEGER;

    PROCEDURE SETNODATA
    (
        aMessage IN VARCHAR(2000)
    )
    IS
    BEGIN
        sNativeError := SQLCODE;
        OUT_mMessage := aMessage;
    END;

    PROCEDURE SETERR
    IS
    BEGIN
        sNativeError := SQLCODE;
        OUT_mMessage := SQLERRM;
    END;

    CURSOR C1 IS SELECT c_id, c_first, c_middle, 
                        c_street_1, c_street_2, c_city, c_state, c_zip, 
                        c_phone, c_credit, c_credit_lim, 
                        c_discount, c_balance, TO_CHAR(c_since,'yyyy-mm-dd hh24:mi:ss')
                   FROM customer 
                  WHERE c_w_id = PM_c_w_id AND 
                        c_d_id = PM_c_d_id AND 
                        c_last = PM_c_last
                  ORDER BY c_first;
BEGIN

    --------------------------------------------------
    -- Input Param, Initialize
    --------------------------------------------------
    PM_w_id      := IN_mW_ID;
    PM_d_id      := IN_mD_ID;
    PM_c_id      := IN_mC_ID;
    PM_c_w_id    := IN_mC_W_ID;
    PM_c_d_id    := IN_mC_D_ID;
    PM_c_last    := IN_mC_LAST;
    PM_h_amount  := IN_mH_AMOUNT;
            
    << RAMP_RETRY >>

    PM_c_data    := '';
    sNativeError := 0;

    --------------------------------------------------
    -- Get Time
    --------------------------------------------------
    PM_h_date := to_char(sysdate, 'yyyy-mm-dd hh24:mi:ss');

    BEGIN
        UPDATE warehouse 
        SET w_ytd = w_ytd + PM_h_amount
        WHERE w_id = PM_w_id
        RETURNING w_street_1,
                  w_street_2,
                  w_city,
                  w_state,
                  w_zip,
                  w_name
        INTO PM_w_street_1,
             PM_w_street_2,
             PM_w_city,
             PM_w_state,
             PM_w_zip,
             PM_w_name;

        IF SQL%ROWCOUNT = 0 THEN SETNODATA('NO DATA FOUND. INVALID W_ID.');goto SQL_FINISH;
        END IF;

    EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
    END;

        
    BEGIN
        UPDATE district
        SET d_ytd = d_ytd + PM_h_amount
        WHERE d_w_id = PM_w_id AND 
              d_id   = PM_d_id
        RETURNING d_street_1, 
                  d_street_2, 
                  d_city, 
                  d_state, 
                  d_zip, 
                  d_name 
        INTO PM_d_street_1,
             PM_d_street_2,
             PM_d_city,
             PM_d_state,
             PM_d_zip,
             PM_d_name;

        IF SQL%ROWCOUNT = 0 THEN SETNODATA('NO DATA FOUND. INVALID D_ID.');goto SQL_FINISH;
        END IF;

    EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
    END;
    
    
    IF ( PM_c_id = 0 )
    THEN
        BEGIN
            SELECT count(c_id) 
            INTO PM_namecnt
            FROM customer 
            WHERE c_last = PM_c_last AND 
                  c_d_id = PM_c_d_id AND 
                  c_w_id = PM_c_w_id ;

            IF PM_namecnt = 0 THEN SETNODATA('NO DATA FOUND. INVALID C_W_ID or C_D_ID or C_LAST.');goto SQL_FINISH;
            END IF;

        EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
        END;

        
        BEGIN
            OPEN C1;

            IF ( MOD( PM_namecnt, 2) <> 0 ) THEN PM_namecnt := PM_namecnt + 1;
            END IF;
        
            FOR i IN 0 .. ( (PM_namecnt / 2 ) - 1)
            LOOP
                FETCH c1 INTO PM_c_id,
                              PM_c_first,
                              PM_c_middle,
                              PM_c_street_1,
                              PM_c_street_2,
                              PM_c_city,
                              PM_c_state,
                              PM_c_zip,
                              PM_c_phone,
                              PM_c_credit,
                              PM_c_credit_lim,
                              PM_c_discount,
                              PM_c_balance,
                              PM_c_since;
                EXIT WHEN C1%NOTFOUND;
            END LOOP;

            CLOSE C1;
        EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
        END;
    ELSE
        BEGIN
            SELECT c_first, c_middle, c_last, 
                   c_street_1, c_street_2, c_city, c_state, c_zip, 
                   c_phone, c_credit, c_credit_lim, 
                   c_discount, c_balance, TO_CHAR(c_since,'yyyy-mm-dd hh24:mi:ss') 
            INTO  PM_c_first,
                  PM_c_middle,
                  PM_c_last,
                  PM_c_street_1,
                  PM_c_street_2,
                  PM_c_city,
                  PM_c_state,
                  PM_c_zip,
                  PM_c_phone,
                  PM_c_credit,
                  PM_c_credit_lim,
                  PM_c_discount,
                  PM_c_balance,
                  PM_c_since
            FROM customer 
            WHERE c_w_id = PM_c_w_id AND 
                  c_d_id = PM_c_d_id AND 
                  c_id   = PM_c_id ;
        EXCEPTION WHEN NO_DATA_FOUND THEN SETNODATA('NO DATA FOUND. INVALID C_W_ID or C_D_ID or C_ID.');goto SQL_FINISH;
                  WHEN OTHERS        THEN SETERR;goto SQL_FINISH;
        END;
    END IF;


    IF ( POSITION ( 'BC' IN PM_c_credit ) <> 0 )
    THEN

        BEGIN
            UPDATE customer 
            SET c_balance = c_balance - PM_h_amount, 
                c_ytd_payment = c_ytd_payment + PM_h_amount, 
                c_payment_cnt = c_payment_cnt + 1, 
                c_data = SUBSTR((TO_CHAR(PM_c_id) || ' ' ||
                                 TO_CHAR(PM_c_d_id) || ' ' ||
                                 TO_CHAR(PM_c_w_id) || ' ' ||
                                 TO_CHAR(PM_d_id) || ' ' ||
                                 TO_CHAR(PM_w_id) || ' ' ||
                                 TO_CHAR(PM_h_amount, '999.99') || ' | ' )
                                 || c_data, 1, 500)
            WHERE c_w_id = PM_c_w_id AND 
                  c_d_id = PM_c_d_id AND 
                  c_id   = PM_c_id
            RETURNING SUBSTR(c_data, 1, 200)
            INTO PM_c_data;

            IF SQL%ROWCOUNT = 0 THEN SETERR;goto SQL_FINISH;
            END IF;

        EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
        END;
    ELSE
        BEGIN
            UPDATE customer 
            SET c_balance = c_balance - PM_h_amount, 
                c_ytd_payment = c_ytd_payment + PM_h_amount, 
                c_payment_cnt = c_payment_cnt + 1 
            WHERE c_w_id = PM_c_w_id AND 
                  c_d_id = PM_c_d_id AND 
                  c_id   = PM_c_id;

            IF SQL%ROWCOUNT = 0 THEN SETERR;goto SQL_FINISH;
            END IF;

        EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
        END;
    END IF;

    BEGIN
        INSERT INTO history 
                    (h_c_d_id, h_c_w_id, h_c_id, h_d_id, 
                     h_w_id, h_date, h_amount, h_data ) 
        VALUES (PM_c_d_id, 
                PM_c_w_id,
                PM_c_id,
                PM_d_id,
                PM_w_id,
                TO_DATE(PM_h_date,'yyyy-mm-dd hh24:mi:ss'),
                PM_h_amount,
                PM_w_name || '    ' || PM_d_name);
    EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
    END;
   

    -----------------------------------------------
    -- Output Param
    -----------------------------------------------
    OUT_mW_STREET_1   := PM_w_STREET_1;
    OUT_mW_STREET_2   := PM_w_STREET_2;
    OUT_mW_CITY       := PM_w_CITY;
    OUT_mW_STATE      := PM_w_STATE;
    OUT_mW_ZIP        := PM_w_ZIP;
    OUT_mH_DATE       := PM_h_date;
    OUT_mD_STREET_1   := PM_d_STREET_1;
    OUT_mD_STREET_2   := PM_d_STREET_2;
    OUT_mD_CITY       := PM_d_CITY;
    OUT_mD_STATE      := PM_d_STATE;
    OUT_mD_ZIP        := PM_d_ZIP;
    OUT_mC_ID         := PM_c_id;
    OUT_mC_FIRST      := PM_c_first;
    OUT_mC_MIDDLE     := PM_c_middle;
    OUT_mC_LAST       := PM_c_last;
    OUT_mC_STREET_1   := PM_c_street_1;
    OUT_mC_STREET_2   := PM_c_street_2;
    OUT_mC_CITY       := PM_c_city;
    OUT_mC_STATE      := PM_c_state;
    OUT_mC_ZIP        := PM_c_zip;
    OUT_mC_PHONE      := PM_c_phone;
    OUT_mC_CREDIT     := PM_c_credit;
    OUT_mC_CREDIT_LIM := PM_c_credit_lim;
    OUT_mC_DISCOUNT   := PM_c_discount;
    OUT_mC_BALANCE    := PM_c_balance;
    OUT_mC_SINCE      := PM_c_since;
    OUT_mC_DATA       := PM_c_data;
    OUT_mSuccess      := 1;
    
    COMMIT;

    goto END_STEP;
 
    << SQL_FINISH >>

    IF C1%ISOPEN IS TRUE
    THEN
        CLOSE c1;
    END IF;

    ROLLBACK;
    
    IF ( sNativeError = -14007 OR sNativeError = -14032 )
    THEN
        goto RAMP_RETRY;
    END IF;

    OUT_mSuccess := 0;

    << END_STEP >>
    NULL;
END;
/

COMMIT;

CREATE OR REPLACE PROCEDURE OrderStatus
(
    IN_mW_ID                 IN     INTEGER,
    IN_mD_ID                 IN     INTEGER,
    IN_mC_ID                 IN     INTEGER,
    IN_mC_LAST               IN     VARCHAR(16),
    OUT_mO_ID               OUT     NATIVE_INTEGER,
    OUT_mC_ID               OUT     NATIVE_INTEGER,
    OUT_mC_FIRST            OUT     VARCHAR(16),
    OUT_mC_MIDDLE           OUT     CHAR(2),
    OUT_mC_LAST             OUT     VARCHAR(16),
    OUT_mO_ENTRY_D          OUT     VARCHAR(21),
    OUT_mC_BALANCE          OUT     NUMERIC(15, 2),
    OUT_mO_CARRIER_ID       OUT     NATIVE_INTEGER,
    OUT_mO_OL_COUNT         OUT     NATIVE_INTEGER,
    OUT_mItemRes            OUT     VARCHAR(4000),
    OUT_mSuccess            OUT     NATIVE_INTEGER,
    OUT_mMessage            OUT     VARCHAR(2000)
)
IS
    sCount                 INTEGER;
    sNativeError           INTEGER;
    
    OS_w_id   INTEGER;
    OS_d_id   INTEGER;
    OS_c_id   INTEGER;

    OS_c_d_id    INTEGER;
    OS_c_w_id    INTEGER;
    OS_c_first   VARCHAR(16);
    OS_c_middle  CHAR(2);
    OS_c_last    VARCHAR(16);
    OS_c_balance NUMERIC(15, 2);
    OS_o_id      INTEGER;

    OS_o_entry_d          VARCHAR(21);
    OS_o_carrier_id       INTEGER;
    OS_ol_i_id            INTEGER;
    OS_ol_supply_w_id     INTEGER;
    OS_ol_quantity        INTEGER;
    OS_ol_amount          NUMERIC(6, 2);
    OS_ol_delivery_d      VARCHAR(21);
    OS_namecnt            INTEGER;

    sOutput  VARCHAR(4000);

    PROCEDURE SETNODATA
    (
        aMessage IN VARCHAR(2000)
    )
    IS
    BEGIN
        sNativeError := SQLCODE;
        OUT_mMessage := aMessage;
    END;

    PROCEDURE SETERR
    IS
    BEGIN
        sNativeError := SQLCODE;
        OUT_mMessage := SQLERRM;
    END;

    CURSOR c1 IS SELECT c_balance, c_first, c_middle, c_id 
                   FROM customer 
                  WHERE c_last = OS_c_last AND 
                        c_d_id = OS_d_id AND 
                        c_w_id = OS_w_id 
                  ORDER BY c_first;

    CURSOR c2 IS SELECT ol_i_id, ol_supply_w_id, ol_quantity,
                        ol_amount, TO_CHAR(ol_delivery_d, 'yyyy-mm-dd')
                   FROM order_line
                  WHERE ol_o_id = OS_o_id AND
                        ol_d_id = OS_d_id AND 
                        ol_w_id = OS_w_id;
BEGIN
    ---------------------------------------------------
    -- Assign Input Param, Initialize PSM Variable.
    ---------------------------------------------------
    OS_w_id      := IN_mW_ID;
    OS_d_id      := IN_mD_ID;
    OS_c_w_id    := IN_mW_ID;
    OS_c_d_id    := IN_mD_ID;
    OS_c_id      := IN_mC_ID;
    OS_c_last    := IN_mC_LAST;

    << RAMP_RETRY >>

    sCount       := 0;
    sNativeError := 0;
    sOutput      := '';

    IF OS_c_id = 0
    THEN
        BEGIN
            SELECT count(c_id) 
            INTO OS_namecnt
            FROM customer 
            WHERE c_last = OS_c_last AND 
                  c_d_id = OS_d_id AND 
                  c_w_id = OS_w_id ;

            IF OS_namecnt = 0 THEN SETNODATA('NO DATA FOUND. INVALID W_ID or D_ID or C_LAST.');goto SQL_FINISH;
            END IF;

        EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
        END;


        BEGIN
            OPEN c1;

            IF ( MOD( OS_namecnt, 2) <> 0 ) THEN OS_namecnt := OS_namecnt + 1;
            END IF;


            FOR i IN 0 .. ( ( OS_namecnt / 2 ) - 1 )
            LOOP
                FETCH c1 INTO OS_c_balance, OS_c_first, OS_c_middle, OS_c_id;
                EXIT WHEN C1%NOTFOUND;
            END LOOP;


            CLOSE c1;
        EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
        END;
    ELSE
        BEGIN
            SELECT c_balance, c_first, c_middle, c_last 
              INTO OS_c_balance,
                   OS_c_first,
                   OS_c_middle,
                   OS_c_last
             FROM customer 
            WHERE c_id   = OS_c_id AND 
                  c_d_id = OS_d_id AND 
                  c_w_id = OS_w_id ;
  
        EXCEPTION WHEN NO_DATA_FOUND THEN SETNODATA('NO DATA FOUND. INVALID W_ID or D_ID or C_ID.');goto SQL_FINISH;
                  WHEN OTHERS        THEN SETERR;goto SQL_FINISH;
        END;
      
    END IF;


    BEGIN
        SELECT /*+ INDEX_DESC(ORDERS, ORDERS_IDX2) */
              o_id,
              CASE WHEN o_carrier_id IS NULL THEN 0 ELSE o_carrier_id END,
              TO_CHAR(o_entry_d, 'yyyy-mm-dd hh24:mi:ss')
        INTO  OS_o_id, OS_o_carrier_id, OS_o_entry_d
        FROM orders 
        WHERE o_w_id = OS_c_w_id AND 
              o_d_id = OS_d_id AND 
              o_c_id = OS_c_id 
        FETCH 1;
    EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
    END;


    BEGIN
        OPEN C2;

        LOOP
            FETCH C2 INTO OS_ol_i_id, OS_ol_supply_w_id, OS_ol_quantity, OS_ol_amount, OS_ol_delivery_d;
            EXIT WHEN C2%NOTFOUND;


            IF ( OS_ol_delivery_d IS NULL ) THEN sOutput := sOutput            ||
                                                            OS_ol_supply_w_id  || '|' ||
                                                            OS_ol_i_id         || '|' || 
                                                            OS_ol_quantity     || '|' ||
                                                            OS_ol_amount       || '|' || '|';
                                            ELSE sOutput := sOutput            ||
                                                            OS_ol_supply_w_id  || '|' ||
                                                            OS_ol_i_id         || '|' || 
                                                            OS_ol_quantity     || '|' ||
                                                            OS_ol_amount       || '|' ||
                                                            OS_ol_delivery_d   || '|';
            END IF;

            sCount := sCount + 1;
                                
        END LOOP;

        CLOSE C2;
    EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
    END;


    ----------------------------------------------
    -- Output
    ----------------------------------------------
    OUT_mO_ID         := OS_o_id;
    OUT_mC_ID         := OS_c_id;
    OUT_mC_FIRST      := OS_c_first;
    OUT_mC_MIDDLE     := OS_c_middle;
    OUT_mC_LAST       := OS_c_last;
    OUT_mO_ENTRY_D    := OS_o_entry_d;
    OUT_mC_BALANCE    := OS_c_balance;
    OUT_mO_CARRIER_ID := OS_o_carrier_id;
    OUT_mO_OL_COUNT   := sCount;
    OUT_mItemRes      := sOutput;
    OUT_mSuccess      := 1;

    COMMIT;
    
    goto END_STEP;
    
    << SQL_FINISH >>

    IF C1%ISOPEN = TRUE
    THEN
        CLOSE c1;
    END IF;

    IF C2%ISOPEN = TRUE
    THEN
        CLOSE c2;
    END IF;

    ROLLBACK;
    
    IF( (sNativeError = -14007 ) OR (sNativeError = -14032 ) ) 
    THEN
        goto RAMP_RETRY;
    END IF;

    OUT_mSuccess := 0;

    << END_STEP >>
    NULL;
END;
/

COMMIT;

CREATE OR REPLACE PROCEDURE Delivery
(
    IN_mW_ID           IN  INTEGER,
    IN_mO_CARRIER_ID   IN  INTEGER,
    OUT_Result         OUT VARCHAR(4000),
    OUT_mSuccess       OUT NATIVE_INTEGER
)
IS
    sNativeError     INTEGER;

    DV_w_id          INTEGER;
    DV_o_carrier_id  INTEGER;
    DV_d_id          INTEGER;
    DV_c_id          INTEGER;
    DV_no_o_id       INTEGER;
    DV_o_total       NUMERIC(15,2);

    sBuffer          VARCHAR(4000);

    PROCEDURE SETERR
    IS
    BEGIN
        sNativeError := SQLCODE;
    END;
BEGIN

    ---------------------------------------------
    -- INPUT ARGS, Initialize Variable
    ---------------------------------------------
    DV_w_id         := IN_mW_ID;
    DV_o_carrier_id := IN_mO_CARRIER_ID;

    << RAMP_RETRY >>

    sNativeError    := 0;
    sBuffer         := '';

    ---------------------------------------------
    -- GET OUTPUT
    ---------------------------------------------
    FOR DV_d_id IN 1 .. 10
    LOOP
        BEGIN
            SELECT no_o_id INTO DV_no_o_id 
            FROM new_order 
            WHERE no_w_id = DV_w_id AND 
                  no_d_id = DV_d_id 
            ORDER BY no_o_id
            FETCH 1 ;
        EXCEPTION WHEN NO_DATA_FOUND THEN sBuffer := sBuffer || -1 || '|' || DV_d_id || '|';
                                          CONTINUE;
                  WHEN OTHERS THEN SETERR;goto SQL_FINISH;
        END;

        BEGIN
            DELETE FROM new_order 
            WHERE no_o_id = DV_no_o_id AND 
                  no_d_id = DV_d_id AND 
                  no_w_id = DV_w_id;
        EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
        END;
       
        BEGIN
            SELECT o_c_id INTO DV_c_id
            FROM orders 
            WHERE o_id   = DV_no_o_id AND 
                  o_d_id = DV_d_id AND
                  o_w_id = DV_w_id;
        EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
        END;

        BEGIN
            UPDATE orders 
            SET o_carrier_id = DV_o_carrier_id
            WHERE o_id   = DV_no_o_id AND 
                  o_d_id = DV_d_id AND 
                  o_w_id = DV_w_id ;
        EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
        END;
    
        BEGIN
            UPDATE order_line 
            SET ol_delivery_d = sysdate
            WHERE ol_o_id = DV_no_o_id AND 
                  ol_d_id = DV_d_id AND 
                  ol_w_id = DV_w_id;
        EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
        END;

        BEGIN
            SELECT SUM(ol_amount) INTO DV_o_total
            FROM order_line 
            WHERE ol_o_id = DV_no_o_id AND 
                  ol_d_id = DV_d_id AND
                  ol_w_id = DV_w_id;
        EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
        END;

        BEGIN
            UPDATE customer 
            SET c_balance = c_balance + DV_o_total, 
                c_delivery_cnt = c_delivery_cnt + 1 
            WHERE c_id   = DV_c_id AND 
                  c_d_id = DV_d_id AND 
                  c_w_id = DV_w_id;
        EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
        END;

        -------------------------------------------
        -- Make Output-String
        -------------------------------------------
        sBuffer := sBuffer || DV_no_o_id || '|' || DV_d_id || '|';
    END LOOP;

   
    -------------------------------------------
    -- Output Param
    -------------------------------------------
    OUT_Result   := sBuffer;
    OUT_mSuccess := 1;

    COMMIT;
    
    goto END_STEP;

    << SQL_FINISH >>

    ROLLBACK;

    IF ( (sNativeError = -14007) OR (sNativeError = -14032) )
    THEN
        goto RAMP_RETRY;
    END IF;

    OUT_mSuccess := 0;

    << END_STEP >>
    NULL;
END;
/

COMMIT;

CREATE OR REPLACE PROCEDURE StockLevel
(
    IN_mW_ID            IN   INTEGER,
    IN_mD_ID            IN   INTEGER,
    IN_mThreshold       IN   INTEGER,
    OUT_O_ID           OUT   NATIVE_INTEGER, 
    OUT_STOCK_COUNT    OUT   NATIVE_INTEGER,
    OUT_mSuccess       OUT   NATIVE_INTEGER,
    OUT_mMessage       OUT   VARCHAR(2000)
)
IS
    sNativeError       INTEGER;

    SL_w_id            INTEGER;
    SL_d_id            INTEGER;
    SL_threshold       INTEGER;
    SL_o_id            INTEGER;
    SL_stock_count     INTEGER; 

    PROCEDURE SETNODATA
    (
        aMessage IN VARCHAR(2000)
    )
    IS
    BEGIN
        sNativeError := SQLCODE;
        OUT_mMessage := aMessage;
    END;

    PROCEDURE SETERR
    IS
    BEGIN
        sNativeError := SQLCODE;
        OUT_mMessage := SQLERRM;
    END;
BEGIN
    
    --------------------------------------------------
    -- Assign Input Param, Initialize PSM Variable
    --------------------------------------------------
    SL_w_id      := IN_mW_ID;
    SL_d_id      := IN_mD_ID;
    SL_threshold := IN_mThreshold;

    << RAMP_RETRY >>
    
    sNativeError := 0;

    BEGIN
        SELECT d_next_o_id 
          INTO SL_o_id
          FROM district 
         WHERE d_w_id = SL_w_id AND 
               d_id   = SL_d_id;
    EXCEPTION WHEN NO_DATA_FOUND THEN SETNODATA('NO DATA FOUND. INVALID W_ID or D_ID.');goto SQL_FINISH;
              WHEN OTHERS        THEN SETERR;goto SQL_FINISH;
    END;

    BEGIN
        SELECT /*+ USE_NL(ORDER_LINE, STOCK) INDEX(ORDER_LINE, ORDER_LINE_PK_IDX) INDEX(STOCK, STOCK_PK_IDX) */ 
               COUNT(DISTINCT(s_i_id)) 
          INTO SL_stock_count
          FROM order_line, stock 
         WHERE ol_w_id = SL_w_id AND 
               ol_d_id = SL_d_id AND 
               ol_o_id < SL_o_id AND 
               ol_o_id >= SL_o_id - 20 AND 
               s_w_id = SL_w_id AND 
               s_i_id = ol_i_id AND 
               s_quantity < SL_threshold ;
    EXCEPTION WHEN OTHERS THEN SETERR;goto SQL_FINISH;
    END;


    --------------------------------------------------
    -- Assign Output Param
    --------------------------------------------------
    OUT_O_ID        := SL_o_id;
    OUT_STOCK_COUNT := SL_stock_count;
    OUT_mSuccess    := 1;

    COMMIT;
    
    goto END_STEP;
    
    << SQL_FINISH >>

    ROLLBACK;
    
    IF ( (sNativeError = -14007) OR (sNativeError = -14032) )
    THEN
        goto RAMP_RETRY;
    END IF;

    OUT_mSuccess := 0;

    << END_STEP >>
    NULL;

END;
/

COMMIT;

