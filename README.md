## Introduction

The goal of this project is to automatically create/update Vulcanproxy rules for all the Docker containers created through Mesos Marathon.

I see many tutorials about how to deploy containers or virtual machines, but I'm always surprised to see that they rarely cover the load balancing part of the infrastructure.

In my opinion, load balancing is a key component of a web scale infrastructure. Why applying automation everywhere if then your application cannot be reached by your users ?

Vulcand is a reverse proxy for HTTP API management and microservices.

And Vulcand is watching etcd to automatically detect new rules it needs to implement, so you don't need to reload any service. Simply add the right keys in etcd and your service/application becomes available from the outside world.

More information available at http://www.vulcanproxy.com

This tool isn't "production ready" and was developped to show end to end automation.

I use Mesos because, even if the native container support has just been added, Mesos is already a robust platform and can be used to deploy other workloads, like Hadoop

### Configuration

Before running this tool, you need to specify for what Marathon applications Vulcanproxy rules should be created.

First, you need to create a root directory in etcd:

```
etcdctl mkdir /mesos-vulcan
```

Then, you need to create a subdirectory using the name of the Marathon app:

```
etcdctl mkdir /mesos-vulcan/app1
```

Finally, you need to indicate the Vulcanproxy frontends and backends to use for this Marathon app

```
etcdctl set /mesos-vulcan/s3pics/frontend f1
etcdctl set /mesos-vulcan/s3pics/backend b1
```

### Run

This tool will:

- determine what Mesos applications are running without a corresponding vulcand rule in etcd and to create the missing rules
- determine what vulcand rules exist in etdc for Mesos applications which aren't running anymore to delete them

The Syntax is pretty simple:

```
./mesos-vulcan -MarathonEndPoint=http://<Marathon instance IP>:8080/v2/apps -EtcdEndPoint=http://<Etcd server IP>:4001/v2 -EtcdRootKey=/mesos-vulcan
```

You can schedule this tool to run every minute to automatically make your Marathon app externally available

# Licensing

Licensed under the Apache License, Version 2.0 (the “License”); you may not use this file except in compliance with the License. You may obtain a copy of the License at <http://www.apache.org/licenses/LICENSE-2.0>

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an “AS IS” BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
