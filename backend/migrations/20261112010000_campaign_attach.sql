-- migrate:up

alter table campaigns
  add column attach text not null default 'none';

alter table campaigns
  add constraint campaigns_attach_check
  check (attach in ('none', 'original', 'render'));

alter table drafts add column attachment_resume_id uuid;
alter table drafts
  add foreign key (workspace_id, attachment_resume_id)
  references resumes (workspace_id, id);

-- migrate:down

alter table drafts drop constraint if exists drafts_workspace_id_attachment_resume_id_fkey;
alter table drafts drop column if exists attachment_resume_id;
alter table campaigns drop constraint if exists campaigns_attach_check;
alter table campaigns drop column if exists attach;
