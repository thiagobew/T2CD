#!/bin/bash

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Step 1: Check and install necessary dependencies
echo "--------------------------------------"
echo "Checking dependencies"
echo "--------------------------------------"
echo " "
# Check for Docker
if ! command_exists docker; then
    echo " "
    echo "--------------------------------------"
    echo "Docker not found. Installing Docker..."
    echo "--------------------------------------"
    echo " "

    sudo apt-get update
    sudo apt-get install -y \
        ca-certificates \
        curl \
        gnupg \
        lsb-release

    sudo mkdir -p /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

    echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
      $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

    sudo apt-get update
    sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
    sudo apt  install docker-compose

    sudo usermod -aG docker $USER
    newgrp docker

    echo "Docker installed successfully."
fi

kubelet kubeadm

# # Check for kubectl
if ! command_exists kubectl; then
    echo " "
    echo "--------------------------------------"
    echo "kubectl not found. Installing kubectl..."
    echo "--------------------------------------"
    echo " "
    sudo apt-get update
    sudo apt-get install -y apt-transport-https ca-certificates curl gpg
    curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.30/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
    echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.30/deb/ /' | sudo tee /etc/apt/sources.list.d/kubernetes.list
    sudo apt-get update
    sudo apt-get install -y kubectl
fi

# # Check for kubeadm
if ! command_exists kubeadm; then
    echo " "
    echo "--------------------------------------"
    echo "kubeadm not found. Installing kubeadm..."
    echo "--------------------------------------"
    echo " "
    sudo apt-get update
    sudo apt-get install -y apt-transport-https ca-certificates curl gpg
    curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.30/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
    echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.30/deb/ /' | sudo tee /etc/apt/sources.list.d/kubernetes.list
    sudo apt-get update
    sudo apt-get install -y kubeadm kubelet
fi

# Hold the applications in their current version
#sudo apt-mark hold kubelet kubeadm kubectl

echo " "
echo "#######################################"
echo "All dependencies are satisfied."
echo "#######################################"
echo " "

# # Step 2: Initialize the Kubernetes cluster using kubeadm
echo " "
echo "--------------------------------------"
echo "Initializing Kubernetes cluster"
echo "--------------------------------------"
echo " "

# Comment out the disabled_plugins line in /etc/containerd/config.toml
sudo sed -i 's/^\(disabled_plugins = \["cri"\]\)$/# \1/' /etc/containerd/config.toml

# Restart the containerd service
sudo systemctl restart containerd.service

# Disable swapoff
sudo swapoff -a
sudo systemctl restart kubelet

sudo kubeadm reset
sudo kubeadm config images list
sudo kubeadm config images pull
#sudo kubeadm init --pod-network-cidr=10.244.0.0/16
sudo kubeadm init --v=5

# Configure kubectl for the non-root user
echo " "
echo "--------------------------------------"
echo "Setting up kubectl for the current user"
echo "--------------------------------------"
echo " "

mkdir -p $HOME/.kube
sudo cp -f /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

# # Step 3: Set up the pod network (using Flannel in this example)
# echo "Setting up the pod network (Flannel)..."
# kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml

# Step 4: Build and run the Go application using Docker Compose
echo " "
echo "--------------------------------------"
echo "Building and running the Go application using Docker Compose"
echo "--------------------------------------"
echo " "

# Define the image name and tag
IMAGE_NAME="docker-tuple-space"
IMAGE_TAG="latest"

# # Check for docker-compose
if ! command_exists docker-compose; then
    echo " "
    echo "--------------------------------------"
    echo "docker-compose not found. Installing docker-compose..."
    echo "--------------------------------------"
    echo " "
    sudo apt  install docker-compose
fi

# Check if the image exists
if [[ "$(docker images -q ${IMAGE_NAME}:${IMAGE_TAG} 2> /dev/null)" == "" ]]; then
  echo "Image ${IMAGE_NAME}:${IMAGE_TAG} not found. Running docker-compose..."
  docker-compose up -d --build

  # Step 5: Deploy the Go application to Kubernetes
  echo " "
  echo "--------------------------------------"
  echo "Deploying the Go application to Kubernetes"
  echo "--------------------------------------"
  echo " "
  # Save the Docker image to a tar file
  docker save docker-tuple-space:latest -o docker-tuple-space.tar

  # Load the Docker image into the Kubernetes cluster
  sudo docker load -i docker-tuple-space.tar
else
  echo " "
  echo "Image ${IMAGE_NAME}:${IMAGE_TAG} already exists."
  echo " "
fi


# Apply the manifests
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml

# Step 6: Verify the deployment
echo " "
echo "--------------------------------------"
echo "Verifying the deployment"
echo "--------------------------------------"
echo " "

# kubectl rollout status deployment/tuple-space-deployment

echo "Your Go application has been successfully deployed and is running in the Kubernetes cluster."

# Clean up the deployment file
#rm deployment.yaml
# rm docker-compose.yml
# rm docker-tuple-space.tar

echo " "
echo "--------------------------------------"
echo "Setup complete!"
echo "--------------------------------------"
echo " "