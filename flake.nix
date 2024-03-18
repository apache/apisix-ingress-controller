{
  description = "APISIX ingress controller";

  inputs = {};
  
  outputs = { self, nixpkgs }: {
    packages.x86_64-linux.default = let
      pkgs = import nixpkgs { system = "x86_64-linux"; };
    in
      pkgs.callPackage ./apisix-ingress-controller.nix {};
  };
}