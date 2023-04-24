-- +goose Up
-- +goose StatementBegin
CREATE TABLE "relayer2_tx" (
  "tx" varchar(255) COLLATE "pg_catalog"."default" NOT NULL,
  "count" int4 NOT NULL DEFAULT 0
);
CREATE TABLE "gaps" (
  "block" int4 NOT NULL
);
ALTER TABLE "gaps" ADD CONSTRAINT "gaps_pkey" PRIMARY KEY ("block");
ALTER TABLE "relayer2_tx" ADD CONSTRAINT "relayer2_tx_pkey" PRIMARY KEY ("tx");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
