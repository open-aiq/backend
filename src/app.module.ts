import { Module } from '@nestjs/common';

import { PrismaModule } from './prisma/prisma.module';
import { UsersModule } from './users/users.module';
import { AppConfigModule } from './app-config/app-config.module';

@Module({
  imports: [PrismaModule, UsersModule, AppConfigModule],
})
export class AppModule {}
