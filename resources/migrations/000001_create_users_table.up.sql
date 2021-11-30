-- format user code: u-randomstring{8}
CREATE TABLE users (
	id serial PRIMARY KEY,
	user_code varchar(10) unique NOT NULL, 
	name varchar(50) NOT NULL,
	email varchar(100) UNIQUE NOT NULL,
	phone varchar(15) UNIQUE NOT null,
	"password" varchar(100) NOT NULL,
	"role" varchar(5) NOT NULL,
	img varchar(200) NULL,
	is_active boolean default false,
	created_date timestamptz(0) NOT NULL,
	updated_date timestamptz(0) NULL,
	deleted_date timestamptz(0) NULL
);