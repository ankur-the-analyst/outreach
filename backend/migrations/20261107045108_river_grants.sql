-- migrate:up

grant select, insert, update, delete on
  river_job, river_leader, river_queue, river_notification, river_migration
  to outreach_app;

grant usage, select, update on
  river_job_id_seq, river_notification_id_seq
  to outreach_app;

-- migrate:down

revoke usage, select, update on
  river_job_id_seq, river_notification_id_seq
  from outreach_app;

revoke select, insert, update, delete on
  river_job, river_leader, river_queue, river_notification, river_migration
  from outreach_app;
