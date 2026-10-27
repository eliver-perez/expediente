CREATE TABLE eula_acceptances (
 id TEXT PRIMARY KEY, installation_id TEXT NOT NULL, user_id TEXT NOT NULL,
 version TEXT NOT NULL, text_sha256 TEXT NOT NULL, accepted_at TEXT NOT NULL,
 FOREIGN KEY(user_id) REFERENCES users(id)
) STRICT;
CREATE INDEX eula_installation_version ON eula_acceptances(installation_id,version,text_sha256);
