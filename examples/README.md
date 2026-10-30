# Sample workflows

These are starting points to copy and adapt, rather than a product registry or managed template catalog. Install the named packages on your Hostlier server before planning a workflow. Substitute your own application image and domain; secret values belong in Hostlier's secret store, never in example files.

- [PostgreSQL app](postgres-app.hostlier): app with a private local PostgreSQL instance.
- [Redis app](redis-app.hostlier): app with a private local Redis instance.
- [Aiven direct](aiven-direct.hostlier): Aiven provider followed by a scoped database operation. Requires explicit provider settings and a stored API token; cloud end-to-end validation remains pending.
- [Legacy Aiven](aiven.hostlier) and [legacy Railway](railway.hostlier): prototype references for the experimental Usefused adapter. These are not supported setup paths.

From a project directory, copy an example to `.hostlier`, edit it, then run:

```sh
hostlier validate --file .hostlier
hostlier plan --file .hostlier
hostlier deploy --file .hostlier
```

The sample app image is a placeholder. Select an application that actually accepts the environment variable exposed by the workflow binding.
