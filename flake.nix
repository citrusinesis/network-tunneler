{
  description = "Go Network Tunneler - L3 Network Tunneling PoC";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        lib = pkgs.lib;

        buildInputs = with pkgs; [
          go
          protobuf
          protoc-gen-go
          protoc-gen-go-grpc
        ];

        buildGoBinary =
          { name, subPackage }:
          pkgs.buildGoModule {
            pname = name;
            version = "0.1.0";
            src = ./.;
            subPackages = [ subPackage ];
            vendorHash = null;
            nativeBuildInputs = buildInputs;
            ldflags = [
              "-s"
              "-w"
            ];
            meta = {
              description = "${name} binary for tunneler test";
              platforms = lib.platforms.all;
            };
          };

        buildDockerImage =
          {
            name,
            binary,
            cmd,
            expose ? [ ],
          }:
          pkgs.dockerTools.buildImage {
            inherit name;
            tag = "latest";
            fromImage = pkgs.dockerTools.pullImage {
              imageName = "ubuntu";
              imageTag = "22.04";
              sha256 = null;
            };
            contents = [
              binary
              pkgs.iproute2
              pkgs.curl
              pkgs.netcat
              pkgs.vim
            ];
            config = {
              Cmd = [
                "/bin/bash"
                "-c"
                cmd
              ];
              ExposedPorts = lib.listToAttrs (
                map (p: {
                  name = "${p}/tcp";
                  value = { };
                }) expose
              );
            };
          };
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs =
            buildInputs
            ++ (
              with pkgs;
              [
                golangci-lint
                gopls
                go-tools
                gotools
                gotests
                delve
                jq
                yq
                gnumake
              ]
              ++ lib.optionals pkgs.stdenv.isLinux [
                iptables
                iproute2
                tcpdump
                wireshark-cli
              ]
            );

          shellHook = ''
            echo "🚀 Go Network Tunneler Development Environment"
            echo "Go version: $(go version)"
            export GOPATH="$PWD/.go"
            export PATH="$GOPATH/bin:$PATH"

            mkdir -p .vscode
            cat > .vscode/settings.json <<EOF
            {
              "go.toolsManagement.autoUpdate": false,
              "go.alternateTools": {
                "go": "$(which go)",
                "gopls": "$(which gopls)",
                "dlv": "$(which dlv)",
                "gotests": "$(which gotests)",
                "staticcheck": "$(which staticcheck)",
                "gofmt": "$(which gofmt)",
                "goimports": "$(which goimports)"
              },
              "go.gopath": "\''${workspaceFolder}/.go",
              "go.useLanguageServer": true,
              "gopls": {
                "formatting.gofumpt": true,
                "ui.semanticTokens": true
              }
            }
            EOF
            echo "✅ VSCode settings configured for Nix Go tools"
          '';
        };

        packages = rec {
          client = buildGoBinary {
            name = "tunneler-client";
            subPackage = "cmd/client";
          };

          server = buildGoBinary {
            name = "tunneler-server";
            subPackage = "cmd/server";
          };

          proxy = buildGoBinary {
            name = "tunneler-proxy";
            subPackage = "cmd/proxy";
          };

          default = pkgs.symlinkJoin {
            name = "tunneler-all";
            paths = [
              client
              server
              proxy
            ];
          };

          docker-client = buildDockerImage {
            name = "tunneler-client";
            binary = client;
            cmd = "${client}/bin/tunneler-client --server server:8080 --cidr 100.64.0.0/10";
          };

          docker-server = buildDockerImage {
            name = "tunneler-server";
            binary = server;
            cmd = "${server}/bin/tunneler-server --listen :8080";
            expose = [ "8080" ];
          };

          docker-proxy = buildDockerImage {
            name = "tunneler-proxy";
            binary = proxy;
            cmd = "${proxy}/bin/tunneler-proxy";
          };
        };
      }
    );
}
