CREATE EXTENSION IF NOT EXISTS citext;
--> statement-breakpoint
CREATE TABLE IF NOT EXISTS "categories" (
	"id" text PRIMARY KEY NOT NULL,
	"plaid_id" text,
	"name" "citext" NOT NULL,
	"user_id" text NOT NULL
);
--> statement-breakpoint
CREATE INDEX IF NOT EXISTS "categories_user_id_idx" ON "categories" ("user_id");--> statement-breakpoint
CREATE UNIQUE INDEX IF NOT EXISTS "categories_user_id_name_uniq" ON "categories" ("user_id","name");