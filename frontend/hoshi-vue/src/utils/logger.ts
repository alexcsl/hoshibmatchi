// Logger utility for Vue application
// Supports multiple log levels based on environment

export enum LogLevel {
  DEBUG = 0,
  INFO = 1,
  WARN = 2,
  ERROR = 3,
  NONE = 4
}

class Logger {
  private level: LogLevel
  private serviceName: string

  constructor(serviceName: string = 'hoshi-vue') {
    this.serviceName = serviceName
    this.level = this.getLogLevelFromEnv()
  }

  private getLogLevelFromEnv(): LogLevel {
    const env = import.meta.env.MODE
    const logLevel = import.meta.env.VITE_LOG_LEVEL

    // Check explicit log level setting
    if (logLevel) {
      switch (logLevel.toUpperCase()) {
        case 'DEBUG':
          return LogLevel.DEBUG
        case 'INFO':
          return LogLevel.INFO
        case 'WARN':
        case 'WARNING':
          return LogLevel.WARN
        case 'ERROR':
          return LogLevel.ERROR
        case 'NONE':
          return LogLevel.NONE
      }
    }

    // Default based on environment
    if (env === 'development' || env === 'dev') {
      return LogLevel.DEBUG
    } else if (env === 'production') {
      return LogLevel.WARN
    }

    return LogLevel.INFO
  }

  setLevel(level: LogLevel): void {
    this.level = level
  }

  getLevel(): LogLevel {
    return this.level
  }

  debug(message: string, ...args: any[]): void {
    if (this.level <= LogLevel.DEBUG) {
      console.log(`[DEBUG] [${this.serviceName}]`, message, ...args)
    }
  }

  info(message: string, ...args: any[]): void {
    if (this.level <= LogLevel.INFO) {
      console.info(`[INFO] [${this.serviceName}]`, message, ...args)
    }
  }

  warn(message: string, ...args: any[]): void {
    if (this.level <= LogLevel.WARN) {
      console.warn(`[WARN] [${this.serviceName}]`, message, ...args)
    }
  }

  error(message: string, ...args: any[]): void {
    if (this.level <= LogLevel.ERROR) {
      console.error(`[ERROR] [${this.serviceName}]`, message, ...args)
    }
  }

  group(label: string): void {
    if (this.level <= LogLevel.DEBUG) {
      console.group(`[${this.serviceName}] ${label}`)
    }
  }

  groupEnd(): void {
    if (this.level <= LogLevel.DEBUG) {
      console.groupEnd()
    }
  }

  table(data: any): void {
    if (this.level <= LogLevel.DEBUG) {
      console.table(data)
    }
  }

  time(label: string): void {
    if (this.level <= LogLevel.DEBUG) {
      console.time(`[${this.serviceName}] ${label}`)
    }
  }

  timeEnd(label: string): void {
    if (this.level <= LogLevel.DEBUG) {
      console.timeEnd(`[${this.serviceName}] ${label}`)
    }
  }
}

// Export the Logger class
export { Logger }

// Export singleton instance
export const logger = new Logger()

// Export factory for creating named loggers
export const createLogger = (serviceName: string): Logger => {
  return new Logger(serviceName)
}

export default logger
