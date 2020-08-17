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

Take your client ID from your Google OAuth2 application credentials, and insert the value into `QUEUE_OAUTH2_CLIENT_ID` in `deploy/docker-compose-dev.yml` or `deploy/docker-compose-prod.yml` depending on your environment (more on that later). You'll also want to change this value in `public/queue.html` in the `google-signin-client_id` `meta` tag.

Finally, ensure `node` is installed on your system, and run `npm install && npm run build`. I'd like to automate this in the future, but we're not directly building it into a container, which makes it a tad difficult. On the plus side, if any changes are made to the JS, another run of `npm run build` will rebuild the bundle and make it immediately available without a container restart.

If you're looking to run a dev environment, that's it! Run `docker-compose -f deploy/docker-compose-dev.yml up -d`, and you're in business (you *might* need to restart the containers the first time you spin them up due to a race condition between the initialization of the database and the application, but once the database is initialized on the first run you shouldn't run into that again). Go to `http://lvh.me:8080` (`lvh.me` always resolves to localhost, but Google OAuth2 requires a domain), and you have a queue! To see the Kibana dashboard, go to `http://kibana.lvh.me:8080`. The default username and password are both `dev`.


### Production

There are a few more steps involved for deploying the production environment. First, go to `deploy/Caddyfile.prod` and change the values of `domain_here` to your domain. When executed, Caddy will automatically fetch TLS certificates for the domain (and subdomain) and keep them renewed through Let's Encrypt. Next, set up a user for the Kibana instance: change `username_here` to a username, and `password_hash_here` to a password hash obtained via `caddy hash-password` (instructions for installing Caddy can be found [here](https://caddyserver.com/docs/download); this doesn't need to be installed in the environment. The hash can be obtained anywhere).

The application can now be started with `docker-compose -f deploy/docker-compose-prod.yml up -d`.

---

Once the application is running, you'll need to drop into the database for one step, which is setting up your email as a site admin. The database is exposed on port 8001 on the host.

```sh
$ psql -h localhost -p 8001 -U queue
queue=# INSERT INTO site_admins (email) VALUES ('your@email.com');
```

From there, you should be able to manage everything from the HTTP API, and shouldn't have to drop into the database. If you do, however, it's always there on port 8001. That's to say: don't expose that port. :)

---

There you go! Make sure ports 80 and 443 are accessible to the host if you're running in production. The queue should be accessible at your domain, and the Kibana instance will be accessible at `kibana.your_domain`, and is password-protected according to the users set up in the `basicauth` directive in `deploy/Caddyfile.prod`.
