-- migrate:up

-- Login must list memberships before a workspace GUC exists.
create policy memberships_by_user on memberships
  using (user_id = nullif(current_setting('app.user_id', true), '')::uuid);

-- Cookie is the session primary key; set app.session_id before SELECT.
create policy sessions_by_id on sessions
  using (id = nullif(current_setting('app.session_id', true), ''));

-- migrate:down

drop policy if exists sessions_by_id on sessions;
drop policy if exists memberships_by_user on memberships;
