-- Mirrors the schema created by the original Rails app, so an existing
-- database can be used without changes.

CREATE TABLE IF NOT EXISTS users (
    id serial PRIMARY KEY,
    email character varying DEFAULT '' NOT NULL,
    encrypted_password character varying DEFAULT '' NOT NULL,
    reset_password_token character varying,
    reset_password_sent_at timestamp without time zone,
    remember_created_at timestamp without time zone,
    sign_in_count integer DEFAULT 0 NOT NULL,
    current_sign_in_at timestamp without time zone,
    last_sign_in_at timestamp without time zone,
    current_sign_in_ip inet,
    last_sign_in_ip inet,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    admin boolean DEFAULT false
);
CREATE UNIQUE INDEX IF NOT EXISTS index_users_on_email ON users (email);
CREATE UNIQUE INDEX IF NOT EXISTS index_users_on_reset_password_token ON users (reset_password_token);

CREATE TABLE IF NOT EXISTS groups (
    id serial PRIMARY KEY,
    name character varying,
    user_id integer REFERENCES users (id),
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);
CREATE INDEX IF NOT EXISTS index_groups_on_user_id ON groups (user_id);

CREATE TABLE IF NOT EXISTS comics (
    id serial PRIMARY KEY,
    filename character varying,
    user_id integer REFERENCES users (id),
    group_id integer REFERENCES groups (id),
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);
CREATE INDEX IF NOT EXISTS index_comics_on_group_id ON comics (group_id);
CREATE INDEX IF NOT EXISTS index_comics_on_user_id ON comics (user_id);

CREATE TABLE IF NOT EXISTS pages (
    id serial PRIMARY KEY,
    number integer,
    read boolean DEFAULT false,
    comic_id integer REFERENCES comics (id),
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    image_file_name character varying,
    image_content_type character varying,
    image_file_size bigint,
    image_updated_at timestamp without time zone
);
CREATE INDEX IF NOT EXISTS index_pages_on_comic_id ON pages (comic_id);

CREATE TABLE IF NOT EXISTS histories (
    id serial PRIMARY KEY,
    user_id integer REFERENCES users (id),
    group_name character varying,
    comic_name character varying,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);
CREATE INDEX IF NOT EXISTS index_histories_on_user_id ON histories (user_id);

CREATE TABLE IF NOT EXISTS sessions (
    token_hash bytea PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at timestamp without time zone NOT NULL,
    created_at timestamp without time zone NOT NULL
);
CREATE INDEX IF NOT EXISTS index_sessions_on_user_id ON sessions (user_id);
