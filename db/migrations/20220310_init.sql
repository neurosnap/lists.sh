CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE app_users (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  email character varying(255) NOT NULL,
  is_verified boolean NOT NULL DEFAULT false,
  created_at timestamp without time zone NOT NULL DEFAULT NOW(),
  updated_at timestamp without time zone NOT NULL DEFAULT NOW(),
  CONSTRAINT app_user_pkey PRIMARY KEY (id),
  CONSTRAINT unique_email UNIQUE (email)
);

CREATE TABLE email_verifications (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  email character varying(255) NOT NULL,
  code character varying(255) NOT NULL,
  expires_at timestamp without time zone NOT NULL,
  used_at timestamp without time zone DEFAULT NULL,
  created_at timestamp without time zone NOT NULL DEFAULT NOW(),
  CONSTRAINT email_verification_pkey PRIMARY KEY (id)
);
CREATE UNIQUE INDEX email_verifications_code ON email_verifications (code);

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
  CONSTRAINT unique_title_for_persona UNIQUE (persona_id, title),
);
