DELETE FROM feature_permissions WHERE feature_key IN (
  'proposals.view','proposals.manage','proposals.accept','proposals.reject',
  'inbox.view','inbox.reply','inbox.manage',
  'tasks.view_own','tasks.view_team','tasks.create_for_self','tasks.create_for_others',
  'tasks.complete','tasks.cancel',
  'copilot.view','copilot.manage','copilot.kill_switch',
  'knowledge.view','knowledge.manage'
);
