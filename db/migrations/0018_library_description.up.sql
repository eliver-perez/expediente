ALTER TABLE libraries ADD COLUMN description TEXT NOT NULL DEFAULT '' CHECK (length(description) <= 500);
