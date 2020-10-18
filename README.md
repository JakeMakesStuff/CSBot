# CSBot
The class Discord bot. This bot probably isn't very useful to you if you're not in my class and contains a lot of in jokes.

## Dev Environment
You have 2 options to develop for this bot locally:

1. Use the provided Dockerfile to develop using Docker.
2. Simply just build the Go project as you normally would.

Note that `TOKEN` is a required env variable to be able to connect to Discord. You must set this for the bot to run.

## SQL Sandbox Deployment Information
The SQL sandbox requires Docker's socket to be forwarded to `/var/run/docker.sock` with the application having rights to be able to interface with it. Additionally, you need to be logged into Docker hub to be able to pull the Oracle database image. You will also require the Oracle Instant Client. This is bundled in the Dockerfile by default.

## Chair Classifier Deployment Information
This requires Cloud Vision to be switched on with your Google Cloud account and your [default credentials](https://cloud.google.com/docs/authentication/production) to be sett.
