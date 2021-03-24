// NOTE must be also eddited in specs.js
requirejs.config({
    baseUrl: "js",
    paths: {
        jquery:     "lib/jquery-1.11.2",
        underscore: "lib/underscore",
        react:      "lib/react",
        CAAT:       "lib/caat",
        wasm:       "app/wasm_exec",
    },

    shim: {
        "underscore": {
            exports: function() {
                return this._.noConflict();
            },
        },
        "wasm": {
            init: function() {
              return new this.Go();
            },
        },
    },

    priority: ["jquery"],
});

require(["client/settings", "wasm"], (settings, go) => {
  WebAssembly.instantiateStreaming(fetch("js/app/app.wasm"), go.importObject).then((result) => {
    go.run(result.instance);

    define("app", ["client/settings"], () => {
      return this.gopherjsApplication.initialize(settings);
    });

    define("github.com/ghthor/filu/rpg2d/coord", ["app"], function(app) {
      return app.coord;
    });

    define("github.com/ghthor/aodd/game", ["app"], function(app) {
      return app.game;
    });

    require([
      "app",
      "ui/login",
    ], function(app, LoginUI) {
      var container = document.getElementById("client");
      app.dial(new LoginUI(container));
    });
  });
});
