CREATE TABLE payments (
	id UUID PRIMARY KEY,
	amount FLOAT NOT NULL,
	processed_at TIMESTAMP NOT NULL,
    processor VARCHAR(20) NOT NULL
);