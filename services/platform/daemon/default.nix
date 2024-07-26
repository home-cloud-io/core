{ lib, buildGoModule, fetchFromGitHub }:

buildGoModule rec {
  pname = "home-cloud-daemon";
  version = "3.1.3";
  vendorHash = "";

  meta = with lib; {
    description = "Home Cloud Host Daemon";
    homepage = "https://github.com/home-cloud-io/core";
    license = licenses.asl20;
    platforms = platforms.linux;
    maintainers = [ maintainers.jgkawell ];
  };

  src = fetchFromGitHub {
    owner = "home-cloud-io";
    repo = "core";
    # rev = "services/platform/daemon/v${version}";
    rev = "feat/daemon-commands";
    hash = "";
  };
}
