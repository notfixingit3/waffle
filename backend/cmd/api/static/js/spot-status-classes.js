var SPOT_BASE_CLASSES = 'admin-spot-item relative rounded-lg border-2 text-center p-2 min-h-[44px] flex flex-col items-center justify-center transition-all duration-200 touch-manipulation select-none';

var SPOT_GRID_CLASSES = Object.freeze({
  available: 'bg-success/10 border-success text-success-content cursor-default',
  pending:   'bg-warning/10 border-warning hover:bg-warning/20 active:bg-warning/30 cursor-pointer text-warning-content',
  paid:      'bg-error/10 border-error text-error-content cursor-default',
  winner:    'bg-secondary/10 border-secondary ring-2 ring-secondary text-secondary-content cursor-default',
  loser:     'bg-base-300 border-base-content/20 text-base-content/50 opacity-60 cursor-default'
});

var SPOT_STATUS_CLASSES = Object.freeze({
  available: Object.freeze({
    bg: 'bg-success',
    text: 'text-success-content',
    ring: 'ring-success',
    badge: 'badge-success',
    border: 'border-success'
  }),
  pending: Object.freeze({
    bg: 'bg-warning',
    text: 'text-warning-content',
    ring: 'ring-warning',
    badge: 'badge-warning',
    border: 'border-warning'
  }),
  paid: Object.freeze({
    bg: 'bg-error',
    text: 'text-error-content',
    ring: 'ring-error',
    badge: 'badge-error',
    border: 'border-error'
  }),
  winner: Object.freeze({
    bg: 'bg-secondary',
    text: 'text-secondary-content',
    ring: 'ring-secondary',
    badge: 'badge-secondary',
    border: 'border-secondary'
  }),
  loser: Object.freeze({
    bg: 'bg-base-300',
    text: 'text-base-content/50',
    ring: 'ring-base-300',
    badge: 'badge-ghost',
    border: 'border-base-300'
  })
});

var SPOT_SELECTION_CLASSES = Object.freeze({
  selected: 'ring-2 ring-primary bg-primary/20',
  bulk_selected: 'ring-2 ring-success bg-success/20',
  hover: 'hover:opacity-90'
});
