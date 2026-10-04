# BabyFileCab vault security status

Implemented in this source:

- A random AES-256-GCM company vault key, wrapped separately for each enrolled user with an Argon2id-derived key. Wrapped keys are stored separately from SQLite.
- New password verifiers use Argon2id; legacy PBKDF2 verifiers upgrade after a successful login.
- Existing company migration is resumable. Before replacement, each plaintext file gets a separately encrypted, verified recovery copy. Encrypted SQLite stores client profiles, user profiles, document descriptions, and calendar entries. The authentication index moves to SQLite when all legacy company profiles have been migrated.
- New uploads, notes, logos, generated forms saved inside the vault, and backups are encrypted; audit records are encrypted SQLite rows. Preview/search and document open decrypt through private temporary files, removed on lock and at the next startup after a crash.
- Existing users stay pending until an administrator with an unlocked vault and the user with their own existing password complete enrollment. The app verifies that user's wrapped key before granting access.
- 15-minute idle locking, lock on loss of window visibility, Linux login1 sleep/lock/session deactivation signals, and a cross-platform wake-gap check.

Limits requiring attention before cloud sharing or production deployment:

- Client names, document filenames, and the folder hierarchy remain visible on the local filesystem. Company names and usernames are visible in the authentication index. Do not upload the local data directory directly to Appwrite.
- The temporary files needed by desktop preview tools and external viewers are plaintext while in use. They are deleted on lock/startup, but filesystem deletion cannot promise to erase old blocks; use full-disk encryption.
- Backups are limited to 512 MB per client. A restored legacy ZIP remains plaintext at its original location. Store or remove it separately.
- Linux has native login1 handling. Windows and macOS rely on window visibility/focus and wake-gap handling; native session events still need platform tests.
- The encryption format and migration have automated tests, but they have not undergone independent security review. Test with a copy of real data and keep a verified separate backup before upgrading.

Run `go test ./...` and `wails dev -tags webkit2_41` on Parrot OS. The frontend JavaScript is a built-in static asset in `frontend/dist`.
