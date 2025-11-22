import { Injectable, Logger, OnModuleInit } from '@nestjs/common';
import { PrismaClient } from 'generated/prisma';

import { DATABASE_CONNECTION_FAILED } from 'src/error-codes.consts';

@Injectable()
export class PrismaService extends PrismaClient implements OnModuleInit {
  private readonly logger = new Logger(PrismaService.name);

  async onModuleInit() {
    try {
      this.logger.log('Connecting to database ...');
      await this.$connect();

      this.logger.log('Connection established with the database.');
    } catch (error) {
      this.logger.error('Prisma connection failed', error);

      process.exit(DATABASE_CONNECTION_FAILED);
    }
  }
}
