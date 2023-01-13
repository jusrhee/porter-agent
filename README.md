# Porter In-Cluster Agent

This repository contains the source code for an in-cluster agent that performs the following operations:
- Provides a query interface for live and historical logs that can be displayed on the Porter dashboard, using Loki as a logging storage backend
- Detects critical incidents on the cluster and forwards alerts to the Porter server
- Stores Kubernetes events and provides a query interface for querying these events (live and historical)

## Codebase Overview

- [pkg/controllers](https://github.com/porter-dev/porter-agent/tree/main/pkg/controllers) contains the three main controllers which can detect events and incidents. These are set up using the Kubernetes informer model
- [pkg/event](https://github.com/porter-dev/porter-agent/blob/main/pkg/event) defines the `FilteredEvent` abstraction and can transform pod-level or event-level events from Kubernetes into filtered events. A filtered event is a subset of events from the Kubernetes API which are considered to have warning or critical severity, and are populated with data such as the Porter release which the triggering pod belongs to
- [pkg/incident](https://github.com/porter-dev/porter-agent/blob/main/pkg/incident) is responsible for determining whether a filtered event should trigger an alert and enumerates all possible incidents/alerts which are sent to the Porter server. 
- [pkg/logstore](https://github.com/porter-dev/porter-agent/tree/main/pkg/logstore) contains the main interface for querying a logging backend. Loki is the only supported logging backend at the moment. 
- [cli](https://github.com/porter-dev/porter-agent/blob/main/cli/main.go) contains a standalone CLI which prints logs to stdout. This is triggered by the Porter API through a `remotecommand` call, so that logs can be streamed back to the dashboard.
