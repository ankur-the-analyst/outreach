-- migrate:up
alter table messages add column body_html text;

-- migrate:down
alter table messages drop column body_html;
