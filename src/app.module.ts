import { Module } from '@nestjs/common';

import { PrismaModule } from './prisma/prisma.module';
import { UsersModule } from './users/users.module';

import { ClerkModule } from './clerk/clerk.module';
import { AppConfigModule } from './config/config.module';

@Module({
  imports: [PrismaModule, UsersModule, AppConfigModule, ClerkModule],
})
export class AppModule {}
