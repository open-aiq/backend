import { Controller, Get } from '@nestjs/common';
import { ApiTags } from '@nestjs/swagger';
import { UsersService } from './users.service';

@ApiTags()
@Controller('users')
export class UsersController {
  constructor(private readonly userService: UsersService) {}
  @Get()
  getUers() {}
}
