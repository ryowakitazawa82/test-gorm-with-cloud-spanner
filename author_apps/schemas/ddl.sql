start batch ddl;

CREATE TABLE if not exists Authors (
  id varchar not null primary key,
  name varchar not null,
  birth_date date,
  created_at timestamptz not null,
  updated_at timestamptz not null
);

CREATE TABLE if not exists Comics (
  id varchar not null primary key,
  name varchar not null,
  created_at timestamptz not null,
  updated_at timestamptz not null
) interleave in parent Authors on delete cascade;

CREATE TABLE if not exists Volumes (
  id varchar not null,
  name varchar not null,
  vol int,
  price int,
  publish_date timestamptz,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  primary key(id, name)
) interleave in parent Comics on delete cascade;

run batch;
