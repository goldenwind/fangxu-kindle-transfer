export function parsePort(value) {
  if (!/^\d+$/.test(String(value).trim())) throw new Error('端口必须是 0 到 65535 的整数 / Port must be an integer from 0 to 65535');
  const port = Number(value);
  if (port > 65535) throw new Error('端口必须是 0 到 65535 的整数 / Port must be an integer from 0 to 65535');
  return port;
}
