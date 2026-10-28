# Day 3 - "The key that was in the repo"

**Stack:** Go (standard library only) - a small notification service
**Tools today:** HashiCorp Vault (L1→L2, new) · Docker (L3) · Docker Compose (L2→L3) · GitHub Actions (L3) · Git (L3) · Linux (L2)
**Time box:** about 4 to 5 hours. Splitting it over two sittings is fine.

## The ticket

> A security review found a live-looking API key and other credentials in this
> service's source and in a `.env` file. The service also prints the key in its
> logs and returns it from an endpoint. Fix the code, move the secret into
> HashiCorp Vault, make sure nothing sensitive ever reaches git or a Docker image,
> and add a pipeline that stops this from happening again.

**Treat every credential in this folder as real,** even though they are fake. That is the habit being trained.

## Carried over (still open from Day 2)
- Day 2's **Jenkins part, non-root user and persistence proof are still open.** Today does not depend on them, so you can finish them before or after. Optional bonus: have Jenkins run today's pipeline too.
- Standing rules: feature branch + PR, `permissions:` block, push-on-`main` + `pull_request` triggers, pinned versions, non-root containers.

## How the service must behave

| Method | Path | Expected behaviour |
|--------|------|--------------------|
| GET | `/health` | `200` `{ "status": "ok" }` |
| GET | `/config/check` | `200` `{ "api_key_loaded": true, "source": "vault" }`. **Never** the key itself |
| POST | `/notify` | body `{ "to": "...", "message": "..." }`. `202` `{ "id", "status": "queued", "key_last4" }`. `400` if `to` or `message` is missing or blank, or the body is not JSON. ids start at 1 |

Startup and secret rules:
- Read `VAULT_ADDR` and `VAULT_TOKEN` from the environment.
- The API key lives in Vault (KV version 2) at `secret/notify`, in the field `api_key`.
- If Vault is unreachable, rejects the token, or the key is missing, the service must **fail to start with a clear error**. No fallback value anywhere.
- Never log or return the secret. Outbound calls must have a timeout.

## Your job

**Part 1 - Fix the app**
- Get `go test ./...` fully green. (Go is installed in a container for you if not on your machine: the official `golang` image works.)
- Read the code as well as the test output. One problem has no test. One test failure hides a more important problem behind it.

**Part 2 - Secrets hygiene in Git**
- Make a **public** repo. Before your first commit, work out which files must never be committed, and add the right ignore rules.
- Provide a `.env.example` with placeholders only.
- Search the whole folder for every place a secret appears and list them in your docs.
- Exercise (in a **throwaway** repo, never your real one): commit a fake secret, delete it in the next commit, then show that it is still in history. Write down what you would do in a real incident.

**Part 3 - Vault**
- Run a Vault **dev** server in Docker (official `hashicorp/vault` image).
- Store the API key at the right path with KV v2.
- Write a **policy** that lets a token read only that secret, and create a token with only that policy. The service must use this token, never the root token.
- Prove it: the scoped token can read `secret/notify` but gets a permission-denied for another path.
- In your docs, explain what is different between dev mode and a production Vault.

**Part 4 - Docker**
- Write a **multi-stage** Dockerfile: build stage with the Go toolchain, tiny final stage, non-root user.
- Write a `.dockerignore`. Prove the final image contains no `.env` and no secret.

**Part 5 - Docker Compose**
- One command starts Vault, loads the secret and policy, and starts the service, in the right order, with healthchecks.
- No token or secret may be committed to git. How you hand the scoped token to the service is your design decision. Explain your choice.
- Call `POST /notify` from outside and show the `202` and `key_last4`.

**Part 6 - GitHub Actions**
- Runs `go vet` and `go test`, with caching where it makes sense.
- Scans the repository for secrets, **including history**. Make it fail when a secret is present. Demonstrate that on a branch.
- Builds the image.
- Carry-over rules apply.

**Stretch**
- In CI, start Vault + the service with Compose and run a real `POST /notify` smoke test.
- **Rotation demo:** write a new value to the secret in Vault, restart the service, and show `key_last4` changes. Show that KV v2 kept the old version.
- Add a pre-commit secret scan locally.

## Done when (send me these)
1. Repo link and PR link(s).
2. `go test` output: red at the start, green at the end.
3. Output proving no secret in the repo or the image.
4. Vault: the policy you wrote, the 200 for your path and the 403 for another path.
5. `docker compose up` output and your `curl` calls (`/health`, `/config/check`, `/notify`).
6. CI: a green run, and a red run caught by the secret scan.
7. Your documentation (template below).

## Documentation template
1. **Scenario** (own words)
2. **Bugs found** - symptom, root cause, fix, and how you found it (test / code reading / logs)
3. **Secrets inventory** - every place a secret appeared, and what you did about each
4. **Git history lesson** - what happens when a secret is committed, and your incident plan
5. **Vault** - setup commands, the policy, token creation, the 200/403 proof, dev vs production
6. **Dockerfile** - each stage and why, how you proved the image is clean
7. **Compose** - startup order, healthchecks, how the token reaches the service and why
8. **CI** - each job and step, including the secret scan
9. **Errors you hit** and how you got unstuck
10. **Clean-up**
11. **3 interview questions** this prepared you for

## Rules
- Stuck for 20 minutes? Docs first, then ask me for **Hint 1**, **2** or **3**.
- You write the Dockerfile, Compose file, Vault policy and workflow yourself.
- Never paste real credentials anywhere: not in the repo, screenshots, chats or LinkedIn. Crop your terminal prompt.
- Vault dev mode is for learning only. Do not expose port 8200 to the internet. Bind to your own machine, or restrict to your IP.
- If you use EC2, stop or terminate it when you finish.
