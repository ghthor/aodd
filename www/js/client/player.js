// TODO This module is 100% unspecified
define(["underscore",
       "client/bar",
       "client/chat_bubble",
       "client/sprite/human",
       "CAAT"
], function(_, Bar, Bubble, Human) {

    var Player = function(params) {
        var player = this;
        var entity = player.entity = params.entity;

        var director = params.director;

        player.createActor = function(name, posX, posY, width, height) {
            var actor = new CAAT.ActorContainer().
                setSize(width, height).
                setPositionAnchored(posX, posY, 0.5, 0.5);

            var animation = new Human(director).
                makeSprites(Human.makeImage(director, "female"), actor.width/2, actor.height/2);
            _.each(animation.sprites, function(sprite) {
                actor.addChild(sprite);
            });

            var nameActor = new CAAT.TextActor().
                setBounds(0, -height, width, height).
                setTextAlign("center").
                setText(name);
            actor.addChild(nameActor);

            var healthBar = new Bar(width, height/4, "red");
            healthBar.actor.
                setPositionAnchored(width/2, -height/2+4, 0.5, 0.5);
            actor.addChild(healthBar.actor);

            player.setHealthPercentage = function(percent) {
                healthBar.setPercent(percent);
            };

            var bubble = new Bubble(150, 80);
            bubble.actor.setPositionAnchored(width/2, -10, 0.5, 1);
            actor.addChild(bubble.actor);

            player.setSayMsg = function(id, msg) {
                bubble.setMsg(id, msg);
            };
            player.clearSayMsg = function(id) {
                bubble.clearMsg(id);
            };


            actor.setAnimation = function(entity) { animation.setAnimation(entity); };
            return actor;
        };

        // Center of the scene
        var center = {
            x: params.scene.width/2,
            y: params.scene.height/2,
        };
        // Tile size
        var gridSz = params.gridSz;

        // A function that sets a movement action on the world's container
        // which makes the player appear to be moving while the sprite
        // stays in the same place on the screen.
        var movePlayer = params.movePlayer;

        // Create the actor when processing the first recieved update
        player.update = function(time, update) {
            var posX = center.x,
                posY = center.y,
                // TODO figure out a better value to use here
                width  = gridSz,
                height = gridSz;

            var actor = player.actor = player.createActor(entity.name, posX, posY, width, height);
            params.scene.addChild(actor);

            player.update = function(time, update) {
                if (!_.isNull(update.pathAction)) {
                    var pathAction = update.pathAction;

                    if (pathAction.start === time) {
                        var duration = pathAction.end - pathAction.start;
                        movePlayer(pathAction.orig, pathAction.dest, duration);
                    }
                }
                player.entity = update;
                actor.setAnimation(update);

                // update health display
                player.setHealthPercentage(update.hp/update.hpMax);
            };
            player.update(time, update);
        };

        return this;
    };

    return Player;
});
