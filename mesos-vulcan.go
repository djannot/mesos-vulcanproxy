package main

import (
  "fmt"
  "bytes"
  "net/http"
  "net/url"
  "github.com/bitly/go-simplejson"
  "strconv"
  "strings"
  "flag"
  )

type Response struct {
  Code int
  Body string
  Headers http.Header
}

type MarathonTask struct {
  Host string
  Port string
}

type MarathonApp struct {
  Id string
  Tasks []MarathonTask
}

var marathonEndPoint string
var etcdEndPoint string
var etcdRootKey string

func main() {
  marathonEndPointPtr := flag.String("MarathonEndPoint", "", "The Marathon API Endpoint")
  etcdEndPointPtr := flag.String("EtcdEndPoint", "", "The Etcd API Endpoint")
  etcdRootKeyPtr := flag.String("EtcdRootKey", "", "The Etcd root key to use for mesos-vulcan")
  flag.Parse()
  marathonEndPoint = *marathonEndPointPtr
  etcdEndPoint = *etcdEndPointPtr
  etcdRootKey = *etcdRootKeyPtr

  marathonApps := make(map[string]MarathonApp)
  // Get all the marathon Apps from the ETCD API
  etcdVulcanResponse, err := httpRequest(etcdEndPoint, "GET", "/keys" + etcdRootKey + "/", nil, "")
  if err != nil {
    fmt.Println(err)
  }
  if etcdVulcanResponse.Code == 200 {
    jsonApps, _ := simplejson.NewJson([]byte(etcdVulcanResponse.Body))
    apps, _ := jsonApps.Get("node").Get("nodes").Array()
    // For each marathon App
    for app, _ := range apps {
      key, _ := jsonApps.Get("node").Get("nodes").GetIndex(app).Get("key").String()
      appId := key[len(etcdRootKey) + 1:]
      // Get information about the marathon App
      marathonAppResponse, _ := httpRequest(marathonEndPoint, "GET", "/" + appId, nil , "")
      jsonApp, _ := simplejson.NewJson([]byte(marathonAppResponse.Body))
      // Get the value of the ETCD backend key corresponding to this marathon App
      etcdGetBackendResponse, err := httpRequest(etcdEndPoint, "GET", "/keys" + etcdRootKey + "/" + appId + "/backend", nil , "")
      if err != nil {
        fmt.Println(err)
      }
      // If the ETCD backend keys corresponding to this marathon App exist
      if etcdGetBackendResponse.Code == 200 {
        jsonBackend, _ := simplejson.NewJson([]byte(etcdGetBackendResponse.Body))
        backend, _ := jsonBackend.Get("node").Get("value").String()
        // Get the value of the ETCD vulcan server keys corresponding to this marathon App
        path := "/keys/vulcand/backends/" + backend + "/servers/"
        etcdGetServersResponse, err := httpRequest(etcdEndPoint, "GET", path , nil , "")
        if err != nil {
          fmt.Println(err)
        }
        if etcdGetServersResponse.Code == 200 {
          jsonServers, _ := simplejson.NewJson([]byte(etcdGetServersResponse.Body))
          servers, _ := jsonServers.Get("node").Get("nodes").Array()
          marathonTasks := []MarathonTask{}
          // For each vulcan server configured for this marathon App
          for server, _ := range servers {
            key, _ := jsonServers.Get("node").Get("nodes").GetIndex(server).Get("key").String()
            directory := key[len(path) - 5:]
            host := directory[:strings.LastIndex(directory, "-")]
            port := directory[strings.LastIndex(directory, "-") + 1:]
            fmt.Println(host, port)
            // Check the container is still running on this host and listening on this port
            containerExist := false
            tasks, _ := jsonApp.Get("app").Get("tasks").Array()
            for task, _ := range tasks {
              // Get the host where the task has been started
              marathonHost, _ := jsonApp.Get("app").Get("tasks").GetIndex(task).Get("host").String()
              // Get the ports the task is listening to
              marathonPorts := []int{}
              items, _ := jsonApp.Get("app").Get("tasks").GetIndex(task).Get("ports").Array()
              for item, _ := range items {
                marathonPort, _ := jsonApp.Get("app").Get("tasks").GetIndex(task).Get("ports").GetIndex(item).Int()
                marathonPorts = append(marathonPorts, marathonPort)
              }
              if host == marathonHost && port == strconv.Itoa(marathonPorts[0]) {
                containerExist = true
              }
            }
            // Delete the ETCD key for the server if the container corresponding to this task of this marathon App isn't running anymore
            if containerExist {
              fmt.Println("Marathon App " + appId + ": The container running on " + host + " and listening on port " + port + " is still running")
              marathonTask := MarathonTask{Host: host, Port: port}
              marathonTasks = append(marathonTasks, marathonTask)
            } else {
              key := "/keys/vulcand/backends/" + backend + "/servers/" + host + "-" + port
              etcdDeleteServerResponse, err := httpRequest(etcdEndPoint, "DELETE", key, nil , "")
              if err != nil {
                fmt.Println(err)
              }
              if etcdDeleteServerResponse.Code == 307 {
                etcdDeleteServerResponse, _ = httpRequest(etcdDeleteServerResponse.Headers["Location"][0], "DELETE", "", nil, "")
                if err != nil {
                  fmt.Println(err)
                }
              }
              if etcdDeleteServerResponse.Code == 200 {
                fmt.Println("Marathon App " + appId + ": The container running on " + host + " and listening on port " + port + " isn't running anymore and the corresponding vulcan server has been deleted")
              } else {
                fmt.Println("Marathon App " + appId + ": The container running on " + host + " and listening on port " + port + " isn't running anymore, but the corresponding vulcan server hasn't been deleted")
              }
            }
          }
          marathonApp := MarathonApp{Id: appId, Tasks: marathonTasks}
          marathonApps[appId] = marathonApp
        }
      }  else {
        fmt.Println("ETCD frontend and backend keys corresponding to the marathon App with the ID " + appId + " aren't configured properly. Can't check the Vulcan servers for this application")
      }
    }
  }

  //marathonApps := make(map[string]MarathonApp)
  // Get all the marathon Apps from the marathon API
  marathonAppsResponse, err := httpRequest(marathonEndPoint, "GET", "/", nil , "")
  if err != nil {
    fmt.Println(err)
  }
  jsonApps, _ := simplejson.NewJson([]byte(marathonAppsResponse.Body))
  apps, _ := jsonApps.Get("apps").Array()
  // For each marathon App
  for app, _ := range apps {
    appId, _ := jsonApps.Get("apps").GetIndex(app).Get("id").String()

    // Get the value of the ETCD frontend and backend keys corresponding to this marathon App
    etcdGetFrontendResponse, err := httpRequest(etcdEndPoint, "GET", "/keys" + etcdRootKey + "/" + appId + "/frontend", nil , "")
    if err != nil {
      fmt.Println(err)
    }
    etcdGetBackendResponse, err := httpRequest(etcdEndPoint, "GET", "/keys" + etcdRootKey + "/" + appId + "/backend", nil , "")
    if err != nil {
      fmt.Println(err)
    }
    // If the ETCD frontend and backend keys corresponding to this marathon App exist
    if etcdGetFrontendResponse.Code == 200 && etcdGetBackendResponse.Code == 200 {
      jsonFrontend, _ := simplejson.NewJson([]byte(etcdGetFrontendResponse.Body))
      jsonBackend, _ := simplejson.NewJson([]byte(etcdGetBackendResponse.Body))
      frontend, _ := jsonFrontend.Get("node").Get("value").String()
      backend, _ := jsonBackend.Get("node").Get("value").String()
      // Get information about the marathon App
      marathonAppResponse, _ := httpRequest(marathonEndPoint, "GET", "/" + appId, nil, "")
      jsonApp, _ := simplejson.NewJson([]byte(marathonAppResponse.Body))
      tasks, _ := jsonApp.Get("app").Get("tasks").Array()
      //marathonTasks := []MarathonTask{}
      // For each task of the marathon App
      for task, _ := range tasks {
        // Get the host where the task has been started
        host, _ := jsonApp.Get("app").Get("tasks").GetIndex(task).Get("host").String()
        // Get the ports the task is listening to
        ports := []int{}
        items, _ := jsonApp.Get("app").Get("tasks").GetIndex(task).Get("ports").Array()
        for item, _ := range items {
          port, _ := jsonApp.Get("app").Get("tasks").GetIndex(task).Get("ports").GetIndex(item).Int()
          ports = append(ports, port)
        }
        // Create the vulcan proxy server entry corresponding to this host and to the first port in ETCD if it's not already defined
        found := false
        for _, marathonApp := range marathonApps {
          if marathonApp.Id == appId {
            for _, marathonTask := range marathonApp.Tasks {
              if marathonTask.Host == host && marathonTask.Port == strconv.Itoa(ports[0]) {
                found = true
              }
            }
          }
        }
        if found == false {
          key := "/keys/vulcand/backends/" + backend + "/servers/" + host + "-" + strconv.Itoa(ports[0])
          value := `value={"Id":"` + host + "-" + strconv.Itoa(ports[0]) + `","URL":"http://` + host + `:` + strconv.Itoa(ports[0]) + `"}`
          headers := make(map[string][]string)
          headers["Content-Type"] = []string{"application/x-www-form-urlencoded"}
          etcdVulcanResponse, _ := httpRequest(etcdEndPoint, "PUT", key, headers, value)
          if err != nil {
            fmt.Println(err)
          }
          if etcdVulcanResponse.Code == 307 {
            etcdVulcanResponse, _ = httpRequest(etcdVulcanResponse.Headers["Location"][0], "PUT", "", headers, value)
            if err != nil {
              fmt.Println(err)
            }
          }
          if etcdVulcanResponse.Code == 200 || etcdVulcanResponse.Code == 201 {
            fmt.Println("Marathon App " + appId + ": Key " + key + " created or updated in ETCD with the value " + value + " for the frontend " + frontend)
          } else {
            fmt.Println("Marathon App " + appId + ": Cannot create the key " + key + " in ETCD with the value " + value + " for the frontend " + frontend)
          }
          //marathonTask := MarathonTask{Host: host, Ports: ports}
          //marathonTasks = append(marathonTasks, marathonTask)
        }
      }
      //marathonApp := MarathonApp{Id: appId, Tasks: marathonTasks}
      //marathonApps[appId] = marathonApp
    // If the ETCD frontend and backend keys corresponding to this marathon App don't exist
    } else {
      fmt.Println("ETCD frontend and backend keys corresponding to the marathon App with the ID " + appId + " aren't configured properly. Can't configure the Vulcan servers for this application")
    }
  }
}

func httpRequest(endPoint string, method string, path string, headers map[string][]string, bodyString string) (Response, error) {
  fullUrl := endPoint + path
  v := url.Values{}
  if len(headers) > 0 {
    v = headers
    fullUrl += "?" + v.Encode()
  }
  httpClient := &http.Client{}
  req, err := http.NewRequest(method, fullUrl, strings.NewReader(bodyString))
  if err != nil {
    return Response{}, err
  }
  req.Header = headers
  resp, err := httpClient.Do(req)
  if err != nil {
    return Response{}, err
  }
  buf := new(bytes.Buffer)
  buf.ReadFrom(resp.Body)
  body := buf.String()
  response := Response{
    Code: resp.StatusCode,
    Body: body,
    Headers: resp.Header,
  }
  return response, nil
}
