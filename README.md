# Docker and Kubernetes: Collatz conjecture

## Overview
This document explains how to write example application which can be deployed using Kubernetes. We should be able to scale some of the components of the application just to get the taste of Kubernetes.
This is what we will cover:

1: Prerequisite

2: Application overview

3: Create/Run standalone application

4: Build the docker images for our components

5: Deploy/Run application using docker-compose

6: Deploy/Run/Scale using Kubernetes

The docker-compose part is optional but we will cover this to see how the application can be deployed just by using docker and docker-compose.

## Prerequisite

1: Install docker, docker-compose and Kubernetes

2: Setup 'go' development environment

3: Create account in github.com and hub.docker.com


## Application overview

We will write an application which will calculate the number of steps for each number as explained in Collatz conjecture
In general, given any number n. if we follow two rules sequentially then the value of n always reaches to 1.

        If n is even, then n = n / 2
  
        If n is odd, then n = 3n + 1

We are not trying to prove or disprove the conjecture (so far nobody could), instead we are simply using the conjecture to demonstrate ho to scale the number of calculations using Kubernetes.
The application will have three components:

1: Consumer

2: Producer

3: Queue

The producer will generate the numbers from 1 to 264 - 1. It will push each of this number into the Queue.
The consumer will fetch the number from the queue and calculate the number of steps required to reach the final outcome which is n == 1.
The queue, we will use RabbitMQ as a queue implementation.

Our final goal is to run one producer, one RabbitMQ server and multiple consumers. We would also like to scale the consumers if we want. The overall design has couple of flaws, we have two single points of failures,
the queue and the producer. May be some day I will fix the design.

## Create/Run standalone application

For this section, you need to fork the repository github.com. For now. we are interested in following files:

consumer/main.go

producer/main.go

shared/shared.go

After the cloning, change the shared/shared.go as follows:

        var RabbitMQUrl = "amqp://guest:guest@rabbitmq:5672/"        
        to
        var RabbitMQUrl = "amqp://guest:guest@localhost:5672/"

Submit your changes to your forked github repository. Alternatively, you can directly edit the shared.go file on github.com. This URL represents the connection URL to RabbitMQ server. Since we are going to run the application in standalone mode, we want our producer and consumer to connect to
RabbitMQ server which will be hosted in localhost. When we move to docker-compose/kubernetes we will revert the changes back to original URL.

In this part we will be running RabbitMQ in docker container (I know, I said we will be running application in standalone mode but it is far easy to setup already configured RabbitMQ server using docker instead of configuring
server by ourselves), we will build the producer and consumer binaries using 'go' compiler and run directly in terminal. At this stage we are only talking about single consumer. However, if we open multiple shells and execute multiple
consumers it will work just fine.

Run the following commands to get the RabbitMQ docker image from hub.docker.com and run the container:

        docker pull rabbitmq

        docker run -d -p 5672:5672 rabbitmq

By default, rabbitmq will listen on port 5672, but it will listen internally in the container. using -p <host_port>:<container_port> we can bind the the container port to the localhost port. When we use URL, localhost:5672 the request will be
forwarded to container's 5672 port.

In the next step, we will build the producer and consumer binaries.
Go to the consumer directory and execute following commands:

        go get -d ./...

        go build

The first command will get all the packages that are referred in consumer/main.go. Specifically, "github.com/icarus3/Collatz/shared" and "github.com/streadway/amqp"
Execute the same commands from the producer directory. After this, we should have 'producer' and 'consumer' binaries in respective directories.

Try, running the ./producer and ./consumer from different shells. You should see producer writing number into the queue and consumer extracting the number from the queue and printing the number of steps in shell.

## Build the docker images for our components.

At some point we want to dockerise the consumer and producer application and run all the three components using docker-compose. To do that we need to build the docker images for producer and consumer.
First, revert the change that you did in shared/shared.go and submit the changes to your github repository. Alternatively, you can directly edit the shared.go file on github.com. When we run all the three images in container,
the RabbitMQ URL localhost:5672 will not work. Since producer and consumer will be running in their own container, we need a way to communicate with third container which is RabbitMQ. This is achieved using services and docker-compose.
In that case, we will refer the RabbitMQ container using the name 'rabbitmq' and our URL becomes rabbitmq:5672. We will see how to do this in following sections. For now we will build the producer and consumer images and push into hub.docker.com

Go to the consumer directory and execute the following commands:

        docker build -t "collatz-consumer" . (Note the '.' at the end of the command, docker build command requires 'Dockerfile' which is in current directory. For more information also, check the content of the this 'Dockerfile')

        docker login --username=yourhubusername

        docker images

After running 'docker images' command, you should see the IMAGE ID for the images that we just created i.e. collatz-consumer. copy that id. and running following commands:

        docker tag <image-id> yourhubusername/collatz-consumer (We need to re-tag the existing image with name 'yourhubusername/collatz-consumer' so that we can push it under our own repository on hub.docker.com)

        docker push yourhubusername/collatz-consumer

Repeat the steps for producer as well. You should see both of these docker images under your repositories in hub.docker.com.

## Deploy/Run application using docker-compose

This is an optional step but may give you some insight on how docker-compose works. Take a look at Collatz/docker-compose.yml
We are creating three services, rabbitmq, consumer and producer with their respective docker images. We are exposing the 5672 port in rabbitmq service and binding with host's 5672 port.

In the consumer and producer section, we have used the 'links' tag and the value as 'rabbitmq'. This is how we are connecting the producer and consumer containers with the rabbitmq container. When we use URL rabbitmq:5672 from the consumer or producer,
we will be communicating with the rabbitmq server. This way, we do not need to know the ip address of the rabbitmq container.

The 'depends_on' tag allow us to set the order in which containers will be started. This tag is not really helpful in our situation, it guarantees that rabbitmq container will be started before consumer and producer, but it does not know whether rabbitmq server
has started or not in that container. To avoid the premature termination of the consumer or producer ( they will try to connect to rabbitmq and if the server is not up and running they will just die and container will be terminated.) I have added retry logic in
producer code and 20 seconds sleep in consumer.

Similarly, 'networks' property also, useless in our situation. The 'networks' allows external containers to be part of the network and communicate with our containers. Our application does not accepts any request from outside world and it will work just by using 'links'.

Go to the 'Collatz' directory and run the command: 'docker-compose up'

It will fetch the respective images from docker hub and configure the containers as mentioned in the docker-compose.yml file. You should see the mix logs in terminal created by producer and consumer.
From another terminal use 'docker-compose down' to bring down all the containers. 

## Deploy/Run/Scale using Kubernetes

Kubernetes deals with pods instead of containers. Each pod may contain one or more containers. In our case we will run single container in each pod. We will create three deployments in Kubernetes, all in single node. (The node can be a virtual or physical machine, set of nodes becomes the kubernetes cluster.). We will talk about two Kubernetes concepts: Deployment and Services.

Deployment allow us to specify the containers images for the node, port mappings and how many replicas we want (among others). Kubernetes uses deployments to orchestrate the pods. Take a look at the kubernetes-consumer-deploy.yml, kubernetes-producer-deploy.yml, kubernetes-rabbitmq-deploy.yml under 'Collatz' directory. 

Services are the abstract way of exposing an application running on set of pods. Services act as a network service. To communicate with the application, requests will be sent to the service then it will forward it to any of the available pods.
It allow us to bind ports to pod ports and send the requests to service using service name/ip and exposed port. There are three types of services we can create: ClusterIP, NodePort and LoadBalancer.
By default, service will be of type ClusterIP, which means that services can talk to each other in same cluster. To accept the request from outside world you will need to create NodePort service. I don't know much about LoadBalancer type.

For our application, we will create service for rabbitmq which will be of type ClusterIP, we will give name 'rabbitmq' to this service so that we can communicate with the service using rabbitmq:5672 URL. Producer and Consumer does not require service. Nobody is sending any request to them instead they will be communicating with the rabbitmq. Take a look at kubernetes-rabbitmq-service.yml under 'Collatz' directory.

The 'selector' property in the yaml file ties the service with the rabbitmq application. To create our deployments and service run the following commands:

        kubectl apply -f  kubernetes-consumer-deploy.yml

        kubectl apply -f kubernetes-producer-deploy.yml

        kubectl apply -f kubernetes-rabbitmq-deploy.yml

        kubectl apply -f kubernetes-rabbitmq-service.yml

To see all the pods, and to get logs of the particular pod run following commands:

        kubectl get pods

        kubectl logs <pod-name>

To scale the consumers, run the following command:

        kubectl scale --replicas=10 deployment <deployment name>
