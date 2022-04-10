CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS app_users (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  created_at timestamp without time zone NOT NULL DEFAULT NOW(),
  CONSTRAINT app_user_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS public_keys (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  owner_id uuid NOT NULL,
  public_key varchar(2048) NOT NULL,
  created_at timestamp without time zone NOT NULL DEFAULT NOW(),
  CONSTRAINT user_public_keys_pkey PRIMARY KEY (id),
  CONSTRAINT unique_key_for_user UNIQUE (owner_id, public_key),
  CONSTRAINT fk_user_public_keys_owner
    FOREIGN KEY(owner_id)
  REFERENCES app_users(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS personas (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  owner_id uuid NOT NULL,
  name character varying(25) NOT NULL,
  created_at timestamp without time zone NOT NULL DEFAULT NOW(),
  CONSTRAINT personas_pkey PRIMARY KEY (id),
  CONSTRAINT unique_name UNIQUE (name),
  CONSTRAINT fk_personas_owner
    FOREIGN KEY(owner_id)
  REFERENCES app_users(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS posts (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  persona_id uuid NOT NULL,
  title character varying(255) NOT NULL,
  text text NOT NULL DEFAULT '',
  publish_at timestamp without time zone NOT NULL DEFAULT NOW(),
  created_at timestamp without time zone NOT NULL DEFAULT NOW(),
  updated_at timestamp without time zone NOT NULL DEFAULT NOW(),
  CONSTRAINT posts_pkey PRIMARY KEY (id),
  CONSTRAINT unique_title_for_persona UNIQUE (persona_id, title),
  CONSTRAINT fk_posts_persona
    FOREIGN KEY(persona_id)
  REFERENCES personas(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);
