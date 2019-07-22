import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';

import { TextSenderComponent } from './components/textSender/textSender.component';
import { VoiceSenderComponent } from './components/voiceSender/voiceSender.component';

const routes: Routes = [
  { path: 'sendText', component: TextSenderComponent },
  { path: 'sendVoice', component: VoiceSenderComponent },
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule]
})
export class AppRoutingModule { }
