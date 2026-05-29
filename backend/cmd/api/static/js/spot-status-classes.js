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
