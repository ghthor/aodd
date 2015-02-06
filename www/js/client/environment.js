define(["underscore"], function(_) {
    // This determines what type of socket is used to send key input
    // browser - WebSocket
    // native  - UDP
    return _.isUndefined(window.nativeBridge) ? "browser" : "native";
});
