package ps

import (
	"time"

	"github.com/auxten/edgeRec/nn"
)

// Trainer is a neural network trainer
type Trainer interface {
	Train(n *nn.Neural, examples, validation Samples, iterations int, shuffle bool)
}

// OnlineTrainer is a basic, online network trainer
type OnlineTrainer struct {
	*internal
	solver    Solver
	printer   *StatsPrinter
	verbosity int
}

// NewTrainer creates a new trainer
func NewTrainer(solver Solver, verbosity int) *OnlineTrainer {
	return &OnlineTrainer{
		solver:    solver,
		printer:   NewStatsPrinter(),
		verbosity: verbosity,
	}
}

type internal struct {
	deltas [][]float64
}

func newTraining(layers []*nn.Layer) *internal {
	deltas := make([][]float64, len(layers))
	for i, l := range layers {
		deltas[i] = make([]float64, len(l.Neurons))
	}
	return &internal{
		deltas: deltas,
	}
}

// Train trains n
func (t *OnlineTrainer) Train(n *nn.Neural, examples, validation Samples, iterations int, shuffle bool) {
	t.internal = newTraining(n.Layers)

	t.printer.Init(n)
	t.solver.Init(n.NumWeights())

	ts := time.Now()
	for i := 1; i <= iterations; i++ {
		if shuffle {
			examples.Shuffle()
		}
		for j := 0; j < len(examples); j++ {
			t.learn(n, examples[j], i)
		}
		if t.verbosity > 0 && i%t.verbosity == 0 && len(validation) > 0 {
			t.printer.PrintProgress(n, validation, time.Since(ts), i)
		}
	}
}

func (t *OnlineTrainer) learn(n *nn.Neural, e Sample, it int) {
	n.Forward(e.Input)
	t.calculateDeltas(n, e.Response)
	t.update(n, it)
}

func (t *OnlineTrainer) Predict(n *nn.Neural, input []float64) []float64 {
	return n.Predict(input)
}

func (t *OnlineTrainer) calculateDeltas(n *nn.Neural, ideal []float64) {
	for i, neuron := range n.Layers[len(n.Layers)-1].Neurons {
		t.deltas[len(n.Layers)-1][i] = nn.GetLoss(n.Config.Loss).Df(
			neuron.Value,
			ideal[i],
			neuron.DActivate(neuron.Value))
	}

	for i := len(n.Layers) - 2; i >= 0; i-- {
		for j, neuron := range n.Layers[i].Neurons {
			var sum float64
			for k, s := range neuron.Out {
				sum += s.Weight * t.deltas[i+1][k]
			}
			t.deltas[i][j] = neuron.DActivate(neuron.Value) * sum
		}
	}
}

func (t *OnlineTrainer) update(n *nn.Neural, it int) {
	var idx int
	for i, l := range n.Layers {
		for j := range l.Neurons {
			for k := range l.Neurons[j].In {
				update := t.solver.Update(l.Neurons[j].In[k].Weight,
					t.deltas[i][j]*l.Neurons[j].In[k].In,
					it,
					idx)
				l.Neurons[j].In[k].Weight += update
				idx++
			}
		}
	}
}
