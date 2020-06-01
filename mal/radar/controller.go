package radar

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/hekmon/hllogger"
	"github.com/hekmon/pushover/v2"
)

const (
	nbSeasonsMin = 1
	nbSeaonsMax  = 40
)

// Config allow to pass configuration when instanciating a new Controller
type Config struct {
	NbSeasons       int
	MinScore        float64
	GenresBlacklist []string
	Pushover        *pushover.Controller
	Logger          *hllogger.HlLogger
}

// New returns an initialized & ready to use controller
func New(ctx context.Context, conf Config) (c *Controller) {
	// config checks
	if conf.Logger == nil {
		panic("can't init mal controller with a nil pushover")
	}
	if conf.Logger == nil {
		panic("can't init mal controller with a nil logger")
	}
	if conf.NbSeasons < nbSeasonsMin {
		conf.Logger.Warningf("[MAL] nbSeasons for initial list building can't be lower than %d (currently: %d): defaulting to %d",
			nbSeasonsMin, conf.NbSeasons, nbSeasonsMin)
		conf.NbSeasons = nbSeasonsMin
	} else if conf.NbSeasons > nbSeaonsMax {
		conf.Logger.Warningf("[MAL] nbSeasons for initial list building can't be more than %d (currently: %d): defaulting to %d",
			nbSeaonsMax, conf.NbSeasons, nbSeaonsMax)
		conf.NbSeasons = nbSeaonsMax
	}
	// create the controller
	c = &Controller{
		ctx:       ctx,
		nbSeasons: conf.NbSeasons,
		minScore:  conf.MinScore,
		blGenres:  conf.GenresBlacklist,
		stopped:   make(chan struct{}),
		pushover:  conf.Pushover,
		log:       conf.Logger,
	}
	if len(c.blGenres) == 0 {
		c.log.Infof("[MAL] controller instanciated with minimum score at %.2f and no blacklisted genre",
			c.minScore)
	} else {
		c.log.Infof("[MAL] controller instanciated with minimum score at %.2f and the following genre(s) blacklisted: %s",
			c.minScore, strings.Join(c.blGenres, ", "))
	}
	// recover previous state if any
	if !c.load(stateFile) {
		c = nil
		return
	}
	c.load(genresFile)
	c.load(ratingsFile)
	// start the worker(s)
	c.workers.Add(1)
	go func() {
		go c.watcher()
		c.workers.Done()
	}()
	// Create the auto-stopper (must be launch after the worker(s) in case ctx is cancelled while launching workers)
	go c.autostop()
	// ready
	return
}

// Controller abstract all the logic of the MAL watcher
type Controller struct {
	// config
	ctx       context.Context
	nbSeasons int
	minScore  float64
	blGenres  []string
	// state
	update    sync.Mutex
	watchList map[int]string
	genres    UniqList
	ratings   UniqList
	// worker(s)
	workers     sync.WaitGroup
	stopped     chan struct{}
	lastRequest time.Time
	// sub controllers
	pushover *pushover.Controller
	log      *hllogger.HlLogger
}

func (c *Controller) autostop() {
	// Wait for signal
	<-c.ctx.Done()
	// Begin the stopping proceedure
	c.workers.Wait()
	// save state
	c.save(stateFile)
	c.save(genresFile)
	c.save(ratingsFile)
	// Close the stopped chan to indicate we are fully stopped
	close(c.stopped)
}

// WaitStopped will block until c is fully stopped.
// To be stopped, c needs to have its context cancelled.
// WaitStopped is safe to be called from multiples goroutines.
func (c *Controller) WaitStopped() {
	<-c.stopped
}

// SaveStateNow permits to save/dump current state to files without stopping the controller
func (c *Controller) SaveStateNow() {
	c.update.Lock()
	c.save(stateFile)
	c.save(genresFile)
	c.save(ratingsFile)
	c.update.Unlock()
}
