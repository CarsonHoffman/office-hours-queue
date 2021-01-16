# Office Hours Queue

An office hours queue featuring "ordered"- (first come, first serve) and "appointments"- (reservations which can be made on the day of) based help queues.

## Deployment

There's a fair bit of initial setup for secrets, after which the deployment can be managed by Docker.

Secrets are stored in the `deploy/secrets` folder; go ahead and create that now:

```sh
$ mkdir deploy/secrets
```

First: in all situations, the back-end needs a key with which to sign the session cookies it creates.

```sh
$ openssl rand 64 > deploy/secrets/signing.key
```

Next, set up the password for the database user. Note the `-n` option, which prevents a trailing newline from being inserted into the file.

```sh
$ echo -n "goodpassword" > deploy/secrets/postgres_password
```

Take your client ID from your Google OAuth2 application credentials, and insert the value into `QUEUE_OAUTH2_CLIENT_ID` in `deploy/docker-compose-dev.yml` or `deploy/docker-compose-prod.yml` depending on your environment (more on that later). You'll also want to insert the client secret in `deploy/secrets/oauth2_client_secret`.

Finally, the queue needs a password with which it controls access to the `/api/metrics` endpoint. Generate a password with:

```sh
$ openssl rand -hex 32 > deploy/secrets/metrics_password
```

You can then set up Prometheus to use basic auth, with username `queue` and the password you just generated, to retrieve statistics about the queue deployment!

To enable certain features like notifications, browsers force the use of HTTPS. To accomplish this, we'll use [`mkcert`](https://github.com/FiloSottile/mkcert), a tool that installs a self-signed certificate authority into the system store and generates certificates with it (that the system will trust). Install it based on the instructions in the tool's README, then navigate to `deploy/secrets`, create a folder called `certs`, navigate into it, then run `mkcert lvh.me` (more on `lvh.me` later). That's it—the server is now running via HTTPS!

Finally, ensure `node` is installed on your system, navigate to the `frontend` directory, and run `npm install && npm run build`. I'd like to automate this in the future, but we're not directly building it into a container, which makes it a tad difficult. On the plus side, if any changes are made to the JS, another run of `npm run build` will rebuild the bundle and make it immediately available without a container restart.

If you're looking to run a dev environment, that's it! Run `docker-compose -f deploy/docker-compose-dev.yml up -d`, and you're in business (you *might* need to restart the containers the first time you spin them up due to a race condition between the initialization of the database and the application, but once the database is initialized on the first run you shouldn't run into that again). Go to `https://lvh.me:8080` (`lvh.me` always resolves to localhost, but Google OAuth2 requires a domain), and you have a queue! To see the Kibana dashboard, go to `https://lvh.me:8080/kibana`. The default username and password are both `dev`.


### Production

There are a few more steps involved for deploying the production environment. First, go to `deploy/Caddyfile.prod` and change `domain_here` to your domain. When executed, Caddy will automatically fetch TLS certificates for the domain and keep them renewed through Let's Encrypt. Next, set up a user for the Kibana instance: change `username_here` to a username, and `password_hash_here` to a password hash obtained via `caddy hash-password` (instructions for installing Caddy can be found [here](https://caddyserver.com/docs/download); this doesn't need to be installed in the environment. The hash can be obtained anywhere).

The application can now be started with `docker-compose -f deploy/docker-compose-prod.yml up -d`.

---

Once the application is running, you'll need to drop into the database for one step, which is setting up your email as a site admin. The database is exposed on port 8001 on the host.

```sh
$ psql -h localhost -p 8001 -U queue
queue=# INSERT INTO site_admins (email) VALUES ('your@email.com');
```

From there, you should be able to manage everything from the HTTP API, and shouldn't have to drop into the database. If you do, however, it's always there on port 8001. That's to say: don't expose that port. :)

---

There you go! Make sure ports 80 and 443 are accessible to the host if you're running in production. The queue should be accessible at your domain, and the Kibana instance will be accessible at `your.domain/kibana`, and is password-protected according to the users set up in the `basicauth` directive in `deploy/Caddyfile.prod`.

# Front-end development

While working on the front-end, it can be annoying to manually re-build for each change. Luckily, Vue supports hot-reload! To take advantage of this, run `npm run serve` in the `frontend` directory, which will run a development server that reloads changes immediately (or: after a few seconds of builds). This development server will proxy requests to the real back-end and change the relevant `Host` and `Origin` headers, so everything should work transparently. The only thing I haven't been able to get working well is logging in on the development server; since it's a different URL the cookies aren't shared with the real instance, and because the redirect URI is set up to go to the real instance, things break down. The solution I've found is to simply copy the `session` cookie from the real back-end's URL and add it to the development server's URL in your browser. This needs to be done each time the session cookie expires (it lasts for 30 days), but this is far better from the old workflow of re-building each time, so it should do.

## Dev Server HTTPS

To enable the use of HTTPS via the dev server, we'll use `mkcert` again. Navigate to `deploy/secrets/certs` and run `mkcert localhost`; the dev server now runs over HTTPS (with a self-signed certificate)!