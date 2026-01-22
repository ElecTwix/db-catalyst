package grammars

const (
	// SQLiteGrammar contains the SQLite DDL grammar rules
	SQLiteGrammar = `
CREATE TABLE [IF NOT EXISTS] [schema_name.]table_name (
  column_def (',' column_def)*
  [',' table_constraint]*
) [WITHOUT ROWID];

column_def ::= column_name data_type column_constraint*;

data_type ::= 
  | INTEGER 
  | TEXT 
  | REAL 
  | BLOB 
  | NUMERIC;

column_constraint ::=
  | PRIMARY KEY [ASC|DESC] [AUTOINCREMENT]
  | NOT NULL
  | UNIQUE
  | CHECK '(' expression ')'
  | DEFAULT default_value
  | COLLATE collation_name
  | REFERENCES table_name [(column_name)] [ON DELETE action] [ON UPDATE action];

table_constraint ::=
  | PRIMARY KEY '(' column_name (',' column_name)* ')'
  | UNIQUE '(' column_name (',' column_name)* ')'
  | FOREIGN KEY '(' column_name (',' column_name)* ')' REFERENCES table_name '(' column_name (',' column_name)* ')' [ON DELETE action] [ON UPDATE action]
  | CHECK '(' expression ')';

action ::= 
  | SET NULL
  | SET DEFAULT
  | CASCADE
  | RESTRICT
  | NO ACTION;
`

	// PostgreSQLGrammar contains the PostgreSQL DDL grammar rules
	PostgreSQLGrammar = `
CREATE [TEMP|TEMPORARY] TABLE [IF NOT EXISTS] table_name (
  column_def (',' column_def)*
  [',' table_constraint]*
);

column_def ::= column_name data_type [column_constraint*];

data_type ::=
  | SMALLINT
  | INTEGER
  | BIGINT
  | DECIMAL [(precision [, scale])]
  | NUMERIC [(precision [, scale])]
  | REAL
  | DOUBLE PRECISION
  | CHAR [(n)]
  | CHARACTER [(n)]
  | VARCHAR [(n)]
  | CHARACTER VARYING [(n)]
  | TEXT
  | BOOLEAN
  | DATE
  | TIME [(precision)] [WITH TIME ZONE]
  | TIMESTAMP [(precision)] [WITH TIME ZONE]
  | JSON
  | JSONB;

column_constraint ::=
  | NOT NULL
  | NULL
  | CHECK '(' expression ')'
  | DEFAULT default_value
  | UNIQUE
  | PRIMARY KEY
  | REFERENCES table_name [(column_name)] [MATCH match_type] [ON DELETE action] [ON UPDATE action];

table_constraint ::=
  | PRIMARY KEY '(' column_name (',' column_name)* ')'
  | UNIQUE '(' column_name (',' column_name)* ')'
  | FOREIGN KEY '(' column_name (',' column_name)* ')' REFERENCES table_name '(' column_name (',' column_name)* ')' [MATCH match_type] [ON DELETE action] [ON UPDATE action]
  | CHECK '(' expression ')'
  | EXCLUDE USING index_method '(' exclude_element WITH operator ')';

match_type ::=
  | FULL
  | PARTIAL
  | SIMPLE;

action ::=
  | NO ACTION
  | RESTRICT
  | CASCADE
  | SET NULL
  | SET DEFAULT;

default_value ::=
  | literal_value
  | expression
  | CURRENT_TIMESTAMP
  | CURRENT_DATE;
`

	// MySQLGrammar contains the MySQL DDL grammar rules
	MySQLGrammar = `
CREATE [TEMPORARY] TABLE [IF NOT EXISTS] table_name (
  column_def (',' column_def)*
  [',' table_constraint]*
) [table_options];

column_def ::= column_name data_type [column_constraint*];

data_type ::=
  | TINYINT [(length)]
  | SMALLINT [(length)]
  | MEDIUMINT [(length)]
  | INT [(length)]
  | INTEGER [(length)]
  | BIGINT [(length)]
  | FLOAT [(precision, scale)]
  | DOUBLE [(precision, scale)]
  | DECIMAL [(precision, scale)]
  | NUMERIC [(precision, scale)]
  | DATE
  | TIME [(fractional_seconds)]
  | DATETIME [(fractional_seconds)]
  | TIMESTAMP [(fractional_seconds)]
  | YEAR
  | CHAR [(length)]
  | VARCHAR [(length)]
  | BINARY [(length)]
  | VARBINARY [(length)]
  | TINYBLOB
  | BLOB
  | MEDIUMBLOB
  | LONGBLOB
  | TINYTEXT
  | TEXT
  | MEDIUMTEXT
  | LONGTEXT
  | ENUM ('value1', 'value2', ...)
  | SET ('value1', 'value2', ...)
  | JSON;

column_constraint ::=
  | NOT NULL
  | NULL
  | DEFAULT default_value
  | AUTO_INCREMENT
  | PRIMARY KEY
  | UNIQUE [KEY]
  | [CONSTRAINT symbol] CHECK '(' expression ')'
  | REFERENCES table_name [(column_name)] [ON DELETE action] [ON UPDATE action];

table_constraint ::=
  | [CONSTRAINT symbol] PRIMARY KEY '(' column_name (',' column_name)* ')'
  | [CONSTRAINT symbol] UNIQUE [INDEX|KEY] '(' column_name (',' column_name)* ')'
  | [CONSTRAINT symbol] FOREIGN KEY '(' column_name (',' column_name)* ')' REFERENCES table_name '(' column_name (',' column_name)* ')' [ON DELETE action] [ON UPDATE action]
  | [CONSTRAINT symbol] CHECK '(' expression ')'
  | [CONSTRAINT symbol] INDEX [index_name] '(' column_name (',' column_name)* ')';

table_options ::=
  | ENGINE [=] engine_name
  | AUTO_INCREMENT [=] value
  | DEFAULT CHARSET [=] charset_name
  | COLLATE [=] collation_name;

action ::=
  | RESTRICT
  | CASCADE
  | SET NULL
  | NO ACTION
  | SET DEFAULT;

default_value ::=
  | literal_value
  | expression
  | CURRENT_TIMESTAMP
  | NOW()
  | NULL;
`
)
