Vagrant.configure("2") do |config|
  config.vm.box = "debian/bookworm64"
  config.vm.box_check_update = false
  
  # Shared provisioning script
  config.vm.provision "shell", inline: <<-SHELL
    apt-get update
    apt-get install -y golang-go build-essential
    mkdir -p /vagrant/build
    cd /vagrant
    go build -o build/nss-daemon cmd/daemon/main.go
    go build -o build/nss-query cmd/query/main.go
    go build -o build/nss-status cmd/status/main.go
    gcc -fPIC -shared -o build/libnss_daemon.so.2 libnss/nss_daemon.c
    install -m 755 build/nss-daemon /usr/local/bin/
    install -m 755 build/nss-query /usr/local/bin/
    install -m 755 build/nss-status /usr/local/bin/
    install -m 644 build/libnss_daemon.so.2 /lib/x86_64-linux-gnu/
    ln -sf /lib/x86_64-linux-gnu/libnss_daemon.so.2 /lib/x86_64-linux-gnu/libnss_daemon.so
    ldconfig
    echo "hosts: files daemon dns" > /etc/nsswitch.conf
  SHELL

  # Node 1
  config.vm.define "node1" do |node|
    node.vm.hostname = "node1"
    node.vm.network :private_network, ip: "192.168.56.10"
    node.vm.provision "shell", inline: <<-SHELL
      cp /vagrant/test/vagrant/config-node1.yaml /etc/nss-daemon/config.yaml
    SHELL
  end

  # Node 2
  config.vm.define "node2" do |node|
    node.vm.hostname = "node2"
    node.vm.network :private_network, ip: "192.168.56.11"
    node.vm.provision "shell", inline: <<-SHELL
      cp /vagrant/test/vagrant/config-node2.yaml /etc/nss-daemon/config.yaml
    SHELL
  end

  # Node 3
  config.vm.define "node3" do |node|
    node.vm.hostname = "node3"
    node.vm.network :private_network, ip: "192.168.56.12"
    node.vm.provision "shell", inline: <<-SHELL
      cp /vagrant/test/vagrant/config-node3.yaml /etc/nss-daemon/config.yaml
    SHELL
  end
end
