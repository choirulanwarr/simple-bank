DROP TRIGGER IF EXISTS audit_customers ON customers;
DROP TRIGGER IF EXISTS audit_accounts ON accounts;
DROP TRIGGER IF EXISTS audit_transactions ON transactions;
DROP TRIGGER IF EXISTS audit_transfers ON transfers;
DROP FUNCTION IF EXISTS audit_trigger_function();