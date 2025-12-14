CREATE TABLE users (
    id bigserial PRIMARY KEY,
    email varchar(255) NOT NULL UNIQUE,
    encrypted_password varchar(255) NOT NULL,
    role varchar(50) NOT NULL DEFAULT 'user',
    token_version int DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = now();
   RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();