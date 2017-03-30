### admin

`admin` is a HTTP server provides the API for the info of alive caches and databases.

We assume that only need one cache master and database master in one oncekv system, so we use the *Default* object using the `config.Config`.

API list:

1. `GET /caches` returns the list of the URL for the current alive cache servers.
1. `Websocket /ws/caches` same reponse data like `/caches`.
1. `GET /dbs` returns the list of the URL for the current alive database servers.
1. `Websocket /ws/dbs` same reponse data like `/dbs`.
