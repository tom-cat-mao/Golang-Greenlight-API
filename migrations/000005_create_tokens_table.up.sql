CREATE TABLE IF NOT EXISTS tokens (
    hash bytea PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    expiry TIMESTAMP(0) with time zone NOT NULL,
    SCOPE text NOT NULL
);