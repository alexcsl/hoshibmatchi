import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { Logger, LogLevel, createLogger } from '../logger'

describe('Logger', () => {
  let consoleSpy: {
    log: ReturnType<typeof vi.spyOn>
    info: ReturnType<typeof vi.spyOn>
    warn: ReturnType<typeof vi.spyOn>
    error: ReturnType<typeof vi.spyOn>
  }

  beforeEach(() => {
    consoleSpy = {
      log: vi.spyOn(console, 'log').mockImplementation(() => {}),
      info: vi.spyOn(console, 'info').mockImplementation(() => {}),
      warn: vi.spyOn(console, 'warn').mockImplementation(() => {}),
      error: vi.spyOn(console, 'error').mockImplementation(() => {}),
    }
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('should create a logger with a service name', () => {
    const logger = createLogger('test-service')
    expect(logger).toBeDefined()
  })

  it('should log debug messages when level is DEBUG', () => {
    const logger = createLogger('test')
    logger.setLevel(LogLevel.DEBUG)
    logger.debug('test message')
    
    expect(consoleSpy.log).toHaveBeenCalledWith(
      '[DEBUG] [test]',
      'test message'
    )
  })

  it('should log info messages when level is INFO or lower', () => {
    const logger = createLogger('test')
    logger.setLevel(LogLevel.INFO)
    logger.info('info message')
    
    expect(consoleSpy.info).toHaveBeenCalledWith(
      '[INFO] [test]',
      'info message'
    )
  })

  it('should log warning messages when level is WARN or lower', () => {
    const logger = createLogger('test')
    logger.setLevel(LogLevel.WARN)
    logger.warn('warning message')
    
    expect(consoleSpy.warn).toHaveBeenCalledWith(
      '[WARN] [test]',
      'warning message'
    )
  })

  it('should log error messages when level is ERROR or lower', () => {
    const logger = createLogger('test')
    logger.setLevel(LogLevel.ERROR)
    logger.error('error message')
    
    expect(consoleSpy.error).toHaveBeenCalledWith(
      '[ERROR] [test]',
      'error message'
    )
  })

  it('should not log debug messages when level is INFO', () => {
    const logger = createLogger('test')
    logger.setLevel(LogLevel.INFO)
    logger.debug('should not appear')
    
    expect(consoleSpy.log).not.toHaveBeenCalled()
  })

  it('should not log info messages when level is WARN', () => {
    const logger = createLogger('test')
    logger.setLevel(LogLevel.WARN)
    logger.info('should not appear')
    
    expect(consoleSpy.info).not.toHaveBeenCalled()
  })

  it('should not log any messages when level is NONE', () => {
    const logger = createLogger('test')
    logger.setLevel(LogLevel.NONE)
    
    logger.debug('should not appear')
    logger.info('should not appear')
    logger.warn('should not appear')
    logger.error('should not appear')
    
    expect(consoleSpy.log).not.toHaveBeenCalled()
    expect(consoleSpy.info).not.toHaveBeenCalled()
    expect(consoleSpy.warn).not.toHaveBeenCalled()
    expect(consoleSpy.error).not.toHaveBeenCalled()
  })

  it('should get and set log level', () => {
    const logger = createLogger('test')
    
    logger.setLevel(LogLevel.ERROR)
    expect(logger.getLevel()).toBe(LogLevel.ERROR)
    
    logger.setLevel(LogLevel.DEBUG)
    expect(logger.getLevel()).toBe(LogLevel.DEBUG)
  })

  it('should log with additional arguments', () => {
    const logger = createLogger('test')
    logger.setLevel(LogLevel.DEBUG)
    
    const data = { key: 'value' }
    logger.debug('message with data', data, 123)
    
    expect(consoleSpy.log).toHaveBeenCalledWith(
      '[DEBUG] [test]',
      'message with data',
      data,
      123
    )
  })
})
