CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE app_users (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  created_at timestamp without time zone NOT NULL DEFAULT NOW(),
  updated_at timestamp without time zone NOT NULL DEFAULT NOW(),
  CONSTRAINT app_user_pkey PRIMARY KEY (id)
);

CREATE TABLE user_public_keys (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  owner_id uuid NOT NULL,
  fingerprint char(64) NOT NULL,
  created_at timestamp without time zone NOT NULL DEFAULT NOW(),
  updated_at timestamp without time zone NOT NULL DEFAULT NOW(),
  CONSTRAINT user_public_keys_pkey PRIMARY KEY (id),
  CONSTRAINT fk_user_public_keys_owner
    FOREIGN KEY(owner_id)
  REFERENCES app_users(id)
  ON DELETE CASCADE
);

CREATE TABLE personas (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  owner_id uuid NOT NULL,
  name character varying(25) NOT NULL,
  created_at timestamp without time zone NOT NULL DEFAULT NOW(),
  updated_at timestamp without time zone NOT NULL DEFAULT NOW(),
  CONSTRAINT personas_pkey PRIMARY KEY (id),
  CONSTRAINT unique_name UNIQUE (name),
  CONSTRAINT fk_personas_owner
    FOREIGN KEY(owner_id)
  REFERENCES app_users(id)
  ON DELETE CASCADE
);

CREATE TABLE posts (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  persona_id uuid NOT NULL,
  title character varying(255) NOT NULL,
  text text NOT NULL DEFAULT '',
  created_at timestamp without time zone NOT NULL DEFAULT NOW(),
  updated_at timestamp without time zone NOT NULL DEFAULT NOW(),
  CONSTRAINT posts_pkey PRIMARY KEY (id),
  CONSTRAINT fk_posts_persona
    FOREIGN KEY(persona_id)
  REFERENCES personas(id)
  ON DELETE CASCADE,
  CONSTRAINT unique_title_for_persona UNIQUE (persona_id, title)
);
