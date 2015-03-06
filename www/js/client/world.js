// TODO This module is 100% unspecified
define(["underscore",
       "client/player",
       "client/bar",
       "client/chat_bubble",
       "client/terrainMap",
       "client/sprite/human",
       "jquery",
       "CAAT"
], function(_, Player, Bar, Bubble, TerrainMap, Human, $) {

    var World = function(director, scene, playerEntity) {
        var world = this;

        world.grid = 16;
        var grid = world.grid,
            width  = 2001 * grid,
            height = 2001 * grid;

        var cellToLocal = world.cellToLocal = function(cell) {
            return {
                x:  width/2 + cell.x * grid,
                y: height/2 - cell.y * grid
            };
        };

        world.scene = scene;

        // This container includes all the entities
        var container = new CAAT.ActorContainer().
            setSize(width, height).
            setPositionAnchored(scene.width/2, scene.height/2, 0.5, 0.5);

        scene.addChild(container);

        TerrainMap.initialize(director);

        // Index of all entities currently being displayed
        var entities = {};
        var actors = {};

        world.move = function(orig, dest, duration) {
            orig = {
               x: -orig.x * grid + scene.width/2,
               y:  orig.y * grid + scene.height/2
            };

            dest = {
               x: -dest.x * grid + scene.width/2,
               y:  dest.y * grid + scene.height/2
            };

            // TODO remove the 40fps hardcoded server fps
            container.emptyBehaviorList().
                addBehavior(new CAAT.PathBehavior().
                    setPath(new CAAT.LinearPath().
                        setInitialPosition(orig.x, orig.y).
                        setFinalPosition(dest.x, dest.y)).
                    setDelayTime(0, duration * 1000.0/40.0));
        };

        world.destroy = function() {
            container.destroy();
        };

        var player = new Player(director, world, playerEntity);

        var createEntityActor = function(entity) {
            var cell = cellToLocal(entity.cell);
            var actor = new CAAT.ActorContainer().
                setSize(grid, grid).
                setPositionAnchored(cell.x, cell.y, 0.5, 0.5);


            var animation = new Human(director).
                makeSprites(Human.makeImage(director, "female"), actor.width/2, actor.height/2);
            _.each(animation.sprites, function(sprite) {
                actor.addChild(sprite);
            });

            var name = new CAAT.TextActor().
                setBounds(0, -grid, grid, grid).
                setTextAlign("center").
                setText(entity.name);
            actor.addChild(name);

            var healthBar = new Bar(grid, grid/4, "red");
            healthBar.actor.
                setPositionAnchored(grid/2, -grid/2+4, 0.5, 0.5);
            actor.addChild(healthBar.actor);

            actor.setHealthPercentage = function(percent) {
                healthBar.setPercent(percent);
            };

            var bubble = new Bubble(150, 80);
            bubble.actor.setPositionAnchored(grid/2, -10, 0.5, 1);
            actor.addChild(bubble.actor);

            actor.setSayMsg = function(id, msg) {
                bubble.setMsg(id, msg);
            };
            actor.clearSayMsg = function(id) {
                bubble.clearMsg(id);
            };

            actor.setAnimation = function(entity) { animation.setAnimation(entity); };
            return actor;
        };

        var actorSetMovement = function(entity, orig, dest, duration) {
            orig = cellToLocal(orig);
            dest = cellToLocal(dest);
            var path = new CAAT.LinearPath().
                setInitialPosition(orig.x, orig.y).
                setFinalPosition(dest.x, dest.y);

            var behavior = new CAAT.PathBehavior().
                setPath(path).
                setDelayTime(0, duration * 1000/40);

            var actor = actors[entity.id];
            actor.emptyBehaviorList();
            actor.addBehavior(behavior);
        };

        var actorUpdatePosition = function(entity) {
            var cell = cellToLocal(entity.cell);
            var actor = actors[entity.id];
            actor.emptyBehaviorList();
            // TODO put this behind a comparision
            actor.setPositionAnchored(cell.x, cell.y, 0.5, 0.5);
        };

        world.entityForId = function(id) {
            if (player.entity.id === id) {
                return player.entity;
            }

            return entities[id];
        };

        world.update = function(update) {
            world.time = update.time;

            // Update all entities
            _.each(update.entities, function(entity) {
                if (entity.id === player.entity.id) {
                    player.update(world.time, entity);
                    return; //continue
                }

                if (!_.isUndefined(entity.type)) {
                    if (entity.type === "assail") {
                        (new Audio("asset/audio/assail.wav")).play();
                    }

                    if (entity.type === "say") {
                        if (entity.saidBy === player.entity.id) {
                            player.setSayMsg(entity.id, entity.msg);
                        } else {
                            actors[entity.saidBy].setSayMsg(entity.id, entity.msg);
                        }
                    }

                    return; //continue
                }

                // Get a handle to the Actor
                var actor;
                if (_.isUndefined(entities[entity.id])) {
                    // Create a new Actor
                    actor = createEntityActor(entity);

                    container.addChild(actor);

                    actors[entity.id] = actor;
                } else {
                    actor = actors[entity.id];
                }

                entities[entity.id] = entity;

                // Check if the entity is moving
                if (!_.isNull(entity.pathAction)) {
                    var pathAction = entity.pathAction;
                    if (pathAction.start === world.time) {
                        var duration = pathAction.end - pathAction.start;
                        actorSetMovement(entity, pathAction.orig, pathAction.dest, duration);
                    }

                } else {
                    actorUpdatePosition(entity);
                }
                
                // Update animation
                actor.setAnimation(entity);

                // update health display
                actor.setHealthPercentage(entity.hp/entity.hpMax);
            });

            // Remove entities that don't exist anymore
            _.each(update.removed, function(entity) {
                if (!_.isUndefined(entity.type)) {
                    if (entity.type === "say") {
                        if (entity.saidBy === player.entity.id) {
                            player.clearSayMsg(entity.id);
                        } else {
                            actors[entity.saidBy].clearSayMsg(entity.id);
                        }
                    }
                    return; //continue
                }

                var actor = actors[entity.id];
                actor.destroy();

                delete entities[entity.id];
                delete actors[entity.id];

                console.log(entity);
            });

            if (!_.isUndefined(update.terrainMap)) {
                world.terrainMap = mergeTerrain(update.terrainMap, world.terrainMap);
            }
        };

        var mergeTerrain = function(map) {
            var canvas = document.createElement("canvas");
            $(canvas).css({
                display: "none",
                position: "absolute",
                top: -9999,
                left: -9999
            });
            document.body.appendChild(canvas);

            map = new TerrainMap(map, canvas, grid);

            var renderedTiles = new CAAT.Actor().setBackgroundImage(canvas, true);
            container.addChildAt(renderedTiles, 0);

            var updatePosition = function(cell) {
                // Moves like an entity in the world
                cell = cellToLocal(cell);
                renderedTiles.setPositionAnchored(cell.x, cell.y, 0.5, 0.5);
            };
            updatePosition(map.center());

            mergeTerrain = function(slice, map) {
                map = map.merge(new TerrainMap(slice));
                updatePosition(map.center());
                return map;
            };
            return map;
        };

        return this;
    };

    return World;
});
