# MyAnimeList Radar

Want to automatically get curated animes as push notifications ? Keep reading :)
Curated animes ? By who ? By the whole MAL community but given ur standards !

## How does it work

MyAnimeList Radar is a small daemon/bot which will monitor MyAnimeList once a day. It will maintain a state of planned/on going animes and once one is finished airing, it will process it in order to determine if you should be notified or not.

### Why only finished animes

* Because I like to binge watch and this was one of the main original reasons to create this bot :3
* Secondly (and more importantly) because it give the community the time to rate them. The more you wait, the more reviews are used to fine grain the final score and the better the anime is curated before processing

### What kind of processing

* First of all the score: you setup a minimal score a finished anime must have to not be ruled out by the bot
* Then you can setup several types of blacklists:
  * Genres blacklist (`Music`, `Kids`, etc...)
  * Types blacklist (`Special`, `Movie`, etc...)
* Finally (this is optionnal) if you have a MAL account, you can specify your username: before each batch of notifications, your profile will be scanned. If an anime about to be notified is present on your list (no matter its status) it won't be notified (because you obviously already know about this one)

### Tell me more about these sweet push notifications

In order to deliver the notifications, an external service is used: [pushover](https://pushover.net/). It allows rich push notifications to be carried to your devices (Android, iOS or even desktop browser). You will need to create an account and an application (more on this a bit later). Keep in mind that while this bot is free, pushover is not. You will need to perform a [$5 USD one-time purchase](https://pushover.net/pricing) for each platform you want to use after the **7 days trial**.

While I understand this could be prohibitive for some, I can assure you this is $5 dollars very well spent: pushover is highly customizable and because you can use your pushover account for a [wide variety of apps](https://pushover.net/apps) and even on your own scripts (the API is very simple) it makes pushover a really nice notifications center for all your projects.

Just to give you an idea, here is 2 (long) screenshots of malradar on pushover, one being the notifications list of the malradar, the other a single notification in open for details:

* [Pushover app listing view](img/list.jpg?raw=true)
* [Pushover app item view](img/item.jpg?raw=true)

## Installation / Configuration

Still interested ? Let's dive in.

### Creation of the pushover app

* First go to [pushover](https://pushover.net/)
  * Create your account
  * Write down your `User Key`
* Then create an [application](https://pushover.net/apps/build)
    * *Name*: Whatever you like, for exemple on my side: `Animes releases`
    * *Description*: again this is just for you to remember what this app is about :) (for the leazy one: `MyAnimeList Radar bot`)
    * *URL*: you can setup `https://github.com/hekmon/malradar` ;)
    * *Icon*: feel free to choose your own, if you have no idea I recommend this [one](https://myanimelist.net/forum/?topicid=1575618) as having no icon is just sad.
    * Once registered, write down the application `API Token`
* Download the app on your phone and log in

### Installation

* MALRadar is prepackaged for Debian-like OSes (this means Ubuntu as well). Simply download the deb package [here](https://github.com/hekmon/malradar/releases) and install it.
  * If you are not on a Debian distribution, you are good for the long run:
    * Setup a working [Golang](https://golang.org/) environment
    * Build MALRadar (`go build`)
    * Take inspiration from the `debian` folder for anything from configuration files to systemd service unit file.
* Edit the configuration file located at `/etc/malradar/config.json` (more details on the next section)
* Start the daemon with `systemctl start malradar.service`

### Configuration

By using the following configuration example:

```json
{
    "myanimelist": {
        "minimum_score": 7.5,
        "user_to_check_against": "",
        "blacklists": {
            "genres": [
                "Hentai",
                "Kids",
                "Music",
                "Shoujo Ai",
                "Shounen Ai",
                "Sports",
                "Yaoi",
                "Yuri"
            ],
            "types": [
                "Special"
            ]
        },
        "initialization": {
            "nb_of_seasons_to_scrape": 4,
            "notify_on_first_run": true
        }
    },
    "pushover": {
        "user_key": "<yourshere>",
        "application_key": "<yourshere>"
    }
}
```

* `myanimelist`
  * `minimum_score`: any anime processed must have at least this score to not be eliminated during the pre notification process
  * `user_to_check_against`: your MAL user. If not empty it will be used to discard any animes already in your list. Particularly usefull for the first run when you have specified a big number of seasons to scan (`nb_of_seasons_to_scrape`) and have not deactivate the initial scan notifications (`notify_on_first_run`).
  * `blacklists`
    * `genres`: if a candidate anime has one or several of these genres, it will be discarded. MALRadar will maintains a list of encountered genres at `/var/lib/malradar/encountered_genres.json` or you can find them [here](https://myanimelist.net/anime.php).
    * `types`: if a candidate anime has its type within this list, it will be discarded. MALRadar will maintains a list of encountered types at `/var/lib/malradar/encountered_types.json`.
  * `initialization`: allow to configure the behavior of MALRadar during first scan
    * `nb_of_seasons_to_scrape`: MALRadar will always start its initial scan for the current season (understand season as 'Summer 2020'). Then it will continue backwards until this number of seasons scanned is reached. High numbers will increase the initial scan duration.
    * `notify_on_first_run`: MALRadar collect already finished animes during the initial scan too. With this parameter you will be notified of all finished animes which pass your processing rules that have aired during the time span configured by `nb_of_seasons_to_scrape`. Usage of the complementary `user_to_check_against` is highly recommended to avoid a notifications flood on the first scan of animes you already know.
* `pushover`
  * `user_key`: the user key you written down earlier
  * `application_key`: the application API key you written down earlier

## State & Backup

MALRadar keeps an internal state to detect animes airing status changes. This state is located at `/var/lib/malradar/animes_state.json` but is only maintained in memory during run. It is saved to disk at stop and loaded from disk at start. But if you want to backup the state without having to stop/backup/start you can issue a `systemctl reload malradar.service` which will safely dump the current in memory state to disk without stopping the bot.

## Third parties

This project would not have been possible without the unofficial MyAnimeList API [jikan](https://jikan.moe/) and the its golang bindings by [darenliang](github.com/darenliang/jikan-go). If you like MALRadar, consider [supporting](https://patreon.com/jikan) the project.
