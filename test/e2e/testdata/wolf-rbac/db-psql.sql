
-- you can change the password on initial the database.
/**
CREATE USER wolfroot WITH PASSWORD '123456';
CREATE DATABASE wolf with owner=wolfroot ENCODING='UTF8';
GRANT ALL PRIVILEGES ON DATABASE wolf to wolfroot;
*/

\c wolf;
create extension pg_trgm;
\c wolf wolfroot;

CREATE FUNCTION unix_timestamp() RETURNS bigint AS $$
SELECT (date_part('epoch',now()))::bigint;
$$ LANGUAGE SQL IMMUTABLE;

CREATE FUNCTION from_unixtime(bigint) RETURNS timestamp AS $$
SELECT to_timestamp($1)::timestamp;
$$ LANGUAGE SQL IMMUTABLE;


CREATE TABLE "application" (
  id text NOT NULL,
  name text NOT NULL,
  "description" text,
  secret text DEFAULT NULL,
  redirect_uris text[] DEFAULT NULL,
  grants text[] DEFAULT NULL,
  access_token_lifetime bigint DEFAULT NULL,
  refresh_token_lifetime bigint DEFAULT NULL,
  create_time bigint NOT NULL,
  update_time bigint NOT NULL,
  primary key(id)
);

CREATE UNIQUE INDEX idx_application_name ON "application"(name);
CREATE INDEX idx_trgm_application_id ON application USING GIN ("id" gin_trgm_ops);
CREATE INDEX idx_trgm_application_name ON application USING GIN ("name" gin_trgm_ops);
COMMENT ON TABLE "application" IS 'Managed applications';
COMMENT ON COLUMN application.id IS 'application id, client.id in oauth2';
COMMENT ON COLUMN application.secret IS 'client.secret in oauth2';
COMMENT ON COLUMN application.redirect_uris IS 'client.redirect_uris in oauth2';
COMMENT ON COLUMN application.access_token_lifetime IS 'access_token.lifetime in oauth2';
COMMENT ON COLUMN application.refresh_token_lifetime IS 'refresh_token.lifetime in oauth2';


CREATE TABLE "user" (
  id bigserial,
  username text not null,
  nickname text,
  email text,
  tel text,
  password text,
  app_ids text[],
  manager text,
  status smallint DEFAULT 0,
  auth_type smallint DEFAULT 1,
  profile jsonb default NULL,
  last_login bigint DEFAULT 0,
  create_time bigint NOT NULL,
  update_time bigint NOT NULL,
  primary key(id)
);

CREATE UNIQUE INDEX idx_user_username ON "user"(username);
CREATE INDEX idx_trgm_user_username ON "user" USING GIN ("username" gin_trgm_ops);
CREATE INDEX idx_trgm_user_nickname ON "user" USING GIN ("nickname" gin_trgm_ops);
CREATE INDEX idx_trgm_user_tel ON "user" USING GIN ("tel" gin_trgm_ops);
CREATE INDEX idx_user_email ON "user"(email);
CREATE INDEX idx_user_app_ids ON "user"(app_ids);
COMMENT ON COLUMN "user".manager IS 'super,admin,NULL';
COMMENT ON COLUMN "user".auth_type IS 'user authentication type, 1: password, 2: LDAP';


CREATE TABLE "category" (
  id serial,
  app_id text NOT NULL,
  name text,
  create_time bigint NOT NULL,
  update_time bigint NOT NULL,
  primary key(id)
);
CREATE UNIQUE INDEX idx_category_app_id_name ON "category"(app_id,name);
CREATE INDEX idx_trgm_category_name ON "category" USING GIN ("name" gin_trgm_ops);


CREATE TABLE "permission" (
  id text,
  app_id text NOT NULL,
  name text NOT NULL,
  "description" text,
  category_id int,
  create_time bigint NOT NULL,
  update_time bigint NOT NULL,
  primary key(app_id, id)
);

CREATE UNIQUE INDEX idx_permission_app_id_name ON "permission"(app_id,name);
CREATE INDEX idx_trgm_permission_id ON "permission" USING GIN ("id" gin_trgm_ops);
CREATE INDEX idx_trgm_permission_name ON "permission" USING GIN ("name" gin_trgm_ops);
CREATE INDEX idx_permission_category_id ON "permission"(category_id);
COMMENT ON COLUMN permission.category_id IS 'reference to category.id';


CREATE TABLE "resource" (
  id bigserial,
  app_id text NOT NULL,
  match_type text NOT NULL,
  name text NOT NULL,
  name_len smallint DEFAULT 0,
  priority bigint DEFAULT 0,
  action text DEFAULT 'ALL',
  perm_id text,
  create_time bigint NOT NULL,
  update_time bigint NOT NULL,
  primary key(id)
);

CREATE UNIQUE INDEX idx_resource_app_id_type_name ON "resource"(app_id,"match_type","name", action);
CREATE INDEX idx_trgm_resource_name ON "resource" USING GIN ("name" gin_trgm_ops);
CREATE INDEX idx_trgm_resource_perm_id ON "resource" USING GIN ("perm_id" gin_trgm_ops);
COMMENT ON COLUMN resource.match_type IS 'The name match type, includes the following:
1. equal, equal match
2. suffix, suffix matching
3. prefix, prefix matching (maximum matching principle)
When matching, equal matches first, if not matched,
Use suffix match, then prefix';
COMMENT ON COLUMN resource.action IS 'for http resource, action is http method: GET, HEAD, POST, OPTIONS, DELETE, PUT, PATCH, ALL means includes all.';


CREATE TABLE "role" (
  id text ,
  app_id text NOT NULL,
  name text NOT NULL,
  "description" text,
  perm_ids text[],
  create_time bigint NOT NULL,
  update_time bigint NOT NULL,
  primary key(app_id, id)
);

CREATE UNIQUE INDEX idx_role_app_id_name ON "role"(app_id, name);
CREATE INDEX idx_trgm_role_id ON "role" USING GIN ("id" gin_trgm_ops);
CREATE INDEX idx_trgm_role_name ON "role" USING GIN ("name" gin_trgm_ops);


CREATE TABLE "user_role" (
  user_id bigint,
  app_id text NOT NULL,
  perm_ids text[],
  role_ids text[],
  create_time bigint NOT NULL,
  update_time bigint NOT NULL,
  primary key(user_id, app_id)
);

CREATE INDEX idx_user_role_perm_ids ON "user_role"(perm_ids);
CREATE INDEX idx_user_role_role_ids ON "user_role"(role_ids);

CREATE TABLE "access_log" (
  id bigserial,
  app_id text,
  user_id text,
  username text,
  nickname text,
  action text,
  res_name text,
  matched_resource jsonb default NULL,
  status smallint DEFAULT 0,
  body jsonb default NULL,
  content_type text,
  date text,
  ip text,
  access_time bigint NOT NULL,
  primary key(id)
);
CREATE INDEX idx_access_log_app_id ON "access_log"(app_id);
CREATE INDEX idx_access_log_user_id ON "access_log"(user_id);
CREATE INDEX idx_access_log_username ON "access_log"(username);
CREATE INDEX idx_access_log_action ON "access_log"(action);
CREATE INDEX idx_access_log_res_name ON "access_log"(res_name);
CREATE INDEX idx_access_log_status ON "access_log"(status);
CREATE INDEX idx_access_log_date ON "access_log"(date);
CREATE INDEX idx_access_log_ip ON "access_log"(ip);
CREATE INDEX idx_access_log_access_time ON "access_log"(access_time);


CREATE TABLE oauth_code (
  id bigserial NOT NULL,
  authorization_code text NOT NULL,
  expires_at timestamp without time zone NOT NULL,
  redirect_uri text NOT NULL,
  scope text DEFAULT NULL,
  client_id text NOT NULL,
  user_id text NOT NULL,
  create_time bigint NOT NULL,
  update_time bigint NOT NULL,
  primary key(id)
);
CREATE UNIQUE INDEX idx_oauth_code_authorization_code ON "oauth_code"(authorization_code);
CREATE INDEX idx_oauth_code_user_id ON "oauth_code"(user_id);
COMMENT ON COLUMN oauth_code.authorization_code IS 'authorization_code in oauth';


CREATE TABLE oauth_token (
  id bigserial NOT NULL,
  access_token text NOT NULL,
  access_token_expires_at timestamp without time zone NOT NULL,
  client_id text NOT NULL,
  refresh_token text,
  refresh_token_expires_at timestamp without time zone,
  scope text DEFAULT NULL,
  user_id text NOT NULL,
  create_time bigint NOT NULL,
  update_time bigint NOT NULL,
  primary key(id)
);
CREATE UNIQUE INDEX idx_oauth_token_access_token ON "oauth_token"(access_token);
CREATE UNIQUE INDEX idx_oauth_token_refresh_token ON "oauth_token"(refresh_token);
CREATE INDEX idx_oauth_token_user_id ON "oauth_token"(user_id);

COMMENT ON COLUMN oauth_token.client_id IS 'client_id in oauth, which corresponds to application.id in this system';
COMMENT ON COLUMN oauth_token.user_id IS 'ID of the user corresponding to client_id, mapped to user.id';
