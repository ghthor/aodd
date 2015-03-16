// TODO rewrite this module as go
define(["underscore",
        "github.com/ghthor/aodd/game",
], function(_, game) {
    var Chat = function(inputConn, pub, entityResolver) {
        var chat = this;

        var time = 0;

        chat.update = function(update) {
            time = update.time;

            _.each(update.entities, function(entity) {
                if (!_.isUndefined(entity.type)) {
                    if (entity.type === "say") {
                        pub.emit("chat/recv/say", [
                                entity.id,
                                entityResolver(entity.saidBy).name,
                                entity.msg,
                                entity.saidAt,
                        ]);
                    }
                }
            });
        };

        chat.sendSay = function(msg) {
            inputConn.sendChatRequest(game.CR_SAY, time, msg);
            pub.emit("chat/sent/say");
        };
    };

    return Chat;
});
