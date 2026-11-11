// Exact argv vectors prevent desktop action arguments from becoming shell commands.
export function actionCommand(action, width = 1280, height = 800) {
  if (!action || typeof action !== 'object' || Array.isArray(action)) throw new Error('expected action object');
  const point = () => [number(action.x, 0, width - 1), number(action.y, 0, height - 1)];
  switch (action.action) {
    case 'move':
      exact(action, ['action', 'x', 'y']);
      return ['mousemove', '--sync', ...point()];
    case 'click': {
      exact(action, ['action', 'x', 'y', 'button']);
      const button = { left: '1', middle: '2', right: '3' }[action.button || 'left'];
      if (!button) throw new Error('invalid button');
      return ['mousemove', '--sync', ...point(), 'click', button];
    }
    case 'type':
      exact(action, ['action', 'text']);
      if (typeof action.text !== 'string' || action.text.length > 16384 || action.text.includes('\0')) throw new Error('invalid text');
      return ['type', '--clearmodifiers', '--delay', '0', '--', action.text];
    case 'key':
      exact(action, ['action', 'key']);
      if (typeof action.key !== 'string' || !/^[A-Za-z0-9_+]{1,128}$/.test(action.key)) throw new Error('invalid key');
      return ['key', '--clearmodifiers', action.key];
    case 'scroll': {
      exact(action, ['action', 'direction', 'amount']);
      const button = { up: '4', down: '5', left: '6', right: '7' }[action.direction];
      if (!button) throw new Error('invalid direction');
      return ['click', '--repeat', number(action.amount ?? 3, 1, 20), '--delay', '30', button];
    }
    default: throw new Error('unknown action');
  }
}

function number(value, min, max) {
  if (!Number.isInteger(value) || value < min || value > max) throw new Error('invalid action coordinate or count');
  return String(value);
}

function exact(action, fields) {
  if (Object.keys(action).some(key => !fields.includes(key))) throw new Error('unknown action field');
}
