## API for Plugins, Jobs and Tasks

> NOTE: This chapter is badly outdated, mainly because it's missing a lot of information. The documenation for individual API calls, however, should be mostly accurate.

**Gopherbot** provides an object-oriented API for writing your own command plugins, jobs and tasks. With the exception of the `bash` library, API calls are accessed from methods on a **robot** object. The following sections detail the usage of the various methods.