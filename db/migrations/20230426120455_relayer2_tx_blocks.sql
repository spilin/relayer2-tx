-- +goose Up
-- +goose StatementBegin
ALTER TABLE relayer2_tx ADD COLUMN blocks integer[];
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
