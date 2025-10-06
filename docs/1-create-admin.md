## Rules for Secure CLI "create admin" Command

### 1. Command Interface
- Command name: e.g.
    `myapp create-admin --email admin@example.com`
- Required: `--email` flag (or positional arg).
- Password: **must not** be passed as a flag or argument (to avoid being visible in shell history, process list, etc.).
### 2. Password Input
- Prompt for password **interactively**:
    - Use `terminal.ReadPassword()` or `golang.org/x/term.ReadPassword()` to read from stdin without echoing the characters.
    - Ask for password **twice** to confirm (avoid typos).
- Example flow:
    `Enter password: ******
  `Confirm password: ******`
- If mismatch → abort with error.
### 3. Security Best Practices
- **Do not log or print** the password anywhere.
- **Do not store plain-text password**. Always hash it:
    - Use `bcrypt` (`golang.org/x/crypto/bcrypt`) with a proper cost factor (default 10–14).
    - Store hash in DB, not the raw password.
- Validate email format before storing.
- Validate password complexity (length, symbols, etc.) if required.

### 4. Error Handling
- Command should fail with clear messages:
    - Missing `--email` flag
    - Passwords don’t match
    - DB errors (duplicate admin, connection issues)
### 5. Example UX

`$ myapp create-admin --email admin@example.com Enter password: ****** Confirm password: ****** ✅ Admin created successfully (email: admin@example.com)`

### 6. Заметки
-