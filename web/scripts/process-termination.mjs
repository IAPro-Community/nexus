import { spawn } from 'node:child_process';

function hasExited(proc) {
  return proc.exitCode !== null || proc.signalCode !== null;
}

function waitForExit(proc, timeoutMs) {
  if (hasExited(proc)) return Promise.resolve(true);

  return new Promise((resolve, reject) => {
    const cleanup = () => {
      clearTimeout(timer);
      proc.removeListener('exit', onExit);
      proc.removeListener('error', onError);
    };
    const onExit = () => {
      cleanup();
      resolve(true);
    };
    const onError = (error) => {
      cleanup();
      reject(error);
    };
    const timer = setTimeout(() => {
      cleanup();
      resolve(false);
    }, timeoutMs);

    proc.once('exit', onExit);
    proc.once('error', onError);
  });
}

function runTaskkill(pid) {
  return new Promise((resolve, reject) => {
    const killer = spawn('taskkill', ['/pid', String(pid), '/T', '/F'], { stdio: 'ignore' });
    killer.once('error', reject);
    killer.once('close', (code) => {
      if (code === 0) {
        resolve();
      } else {
        reject(new Error(`taskkill failed for PID ${pid} with exit code ${code}`));
      }
    });
  });
}

export async function terminateProcess(proc) {
  if (!proc || !proc.pid || hasExited(proc)) return;

  if (process.platform === 'win32') {
    try {
      await runTaskkill(proc.pid);
    } catch (error) {
      if (!(await waitForExit(proc, 5000))) throw error;
      return;
    }
    if (!(await waitForExit(proc, 5000))) {
      throw new Error(`Process ${proc.pid} did not exit after taskkill`);
    }
    return;
  }

  try {
    process.kill(-proc.pid, 'SIGTERM');
  } catch (error) {
    if (error.code === 'ESRCH' && (await waitForExit(proc, 1000))) return;
    throw error;
  }

  if (await waitForExit(proc, 1200)) return;

  try {
    process.kill(-proc.pid, 'SIGKILL');
  } catch (error) {
    if (error.code !== 'ESRCH') throw error;
  }
  if (!(await waitForExit(proc, 5000))) {
    throw new Error(`Process ${proc.pid} did not exit after SIGKILL`);
  }
}
